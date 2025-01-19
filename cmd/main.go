package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jamolkhon5/mistral/internal/auth"
	"github.com/Jamolkhon5/mistral/internal/config"
	"github.com/Jamolkhon5/mistral/internal/handler"
	"github.com/Jamolkhon5/mistral/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Starting Mistral Chat service...")

	// Загрузка конфигурации
	cfg, err := config.NewConfig(".env")
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к базе данных
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PgHost, cfg.PgPort, cfg.PgUser, cfg.PgPassword, cfg.PgName)

	// Ожидание готовности базы данных
	db, err := waitForDatabase(dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создание таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			role VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database initialized successfully")

	// Подключение к сервису auth
	authConfig, err := auth.NewConfig(".auth.env")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := grpc.Dial(authConfig.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Println("Connected to auth service")

	auth.InitClient(conn)

	// Инициализация репозитория и хендлера
	repo := repository.NewRepository(db)
	handler := handler.NewHandler(repo, cfg.MistralApiKey, cfg.ModelName)

	// Настройка роутера
	r := chi.NewRouter()

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Добавьте другие разрешенные origins при необходимости
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Middleware
	r.Use(middleware.Logger)                    // Логирование запросов
	r.Use(middleware.Recoverer)                 // Восстановление после паник
	r.Use(middleware.Timeout(60 * time.Second)) // Таймаут запросов
	r.Use(middleware.RealIP)                    // Получение реального IP
	r.Use(middleware.RequestID)                 // Добавление ID запроса
	r.Use(middleware.CleanPath)                 // Очистка пути
	r.Use(middleware.StripSlashes)              // Удаление слешей в конце

	// Routes
	r.Route("/v1", func(r chi.Router) {
		// Добавляем middleware для проверки авторизации
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		// Основные эндпоинты
		r.Post("/chat", handler.Chat)
		r.Post("/clear-history", handler.ClearHistory)

		// Добавляем health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	})

	// Настройка и запуск сервера
	srv := &http.Server{
		Addr:         ":5641",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server is starting on port %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	// Graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exited properly")
}

// Функция ожидания готовности базы данных
func waitForDatabase(dbURL string) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	maxAttempts := 30

	for i := 0; i < maxAttempts; i++ {
		log.Printf("Attempting to connect to database (attempt %d/%d)...\n", i+1, maxAttempts)
		db, err = sqlx.Connect("postgres", dbURL)
		if err == nil {
			log.Println("Successfully connected to database")
			return db, nil
		}
		log.Printf("Failed to connect: %v. Retrying in 2 seconds...\n", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %v", maxAttempts, err)
}
