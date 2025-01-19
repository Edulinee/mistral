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

	projectAI "github.com/Jamolkhon5/mistral/internal/ai/project/handler"
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
	// Настройка логирования
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("Запуск сервиса Mistral Chat...")

	// Загрузка основной конфигурации
	cfg, err := config.NewConfig(".env")
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации:", err)
	}

	// Подключение к базе данных
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PgHost, cfg.PgPort, cfg.PgUser, cfg.PgPassword, cfg.PgName)

	// Ожидание готовности базы данных
	db, err := waitForDatabase(dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	defer db.Close()

	// Инициализация таблиц
	if err := initializeTables(db); err != nil {
		log.Fatal("Ошибка инициализации таблиц:", err)
	}
	log.Println("База данных успешно инициализирована")

	// Подключение к сервису аутентификации
	authConn, err := initializeAuthService()
	if err != nil {
		log.Fatal("Ошибка подключения к сервису аутентификации:", err)
	}
	defer authConn.Close()
	log.Println("Подключено к сервису аутентификации")

	// Инициализация репозитория и обработчиков
	repo := repository.NewRepository(db)
	chatHandler := handler.NewHandler(repo, cfg.MistralApiKey, cfg.ModelName)
	projectAssistant := projectAI.NewProjectAssistantHandler(cfg.MistralApiKey, cfg.ModelName)

	// Настройка роутера
	router := setupRouter()

	// Регистрация маршрутов
	registerRoutes(router, chatHandler, projectAssistant)

	// Настройка и запуск сервера
	server := setupServer(router)

	// Запуск сервера в горутине
	go startServer(server)

	// Ожидание сигнала для graceful shutdown
	waitForShutdown(server)
}

func waitForDatabase(dbURL string) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	maxAttempts := 30

	for i := 0; i < maxAttempts; i++ {
		log.Printf("Попытка подключения к базе данных (%d/%d)...", i+1, maxAttempts)
		db, err = sqlx.Connect("postgres", dbURL)
		if err == nil {
			log.Println("Успешное подключение к базе данных")
			return db, nil
		}
		log.Printf("Ошибка подключения: %v. Повторная попытка через 2 секунды...", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("не удалось подключиться к базе данных после %d попыток: %v", maxAttempts, err)
}

func initializeTables(db *sqlx.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS messages (
            id SERIAL PRIMARY KEY,
            user_id VARCHAR(255) NOT NULL,
            message TEXT NOT NULL,
            role VARCHAR(50) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS project_conversations (
            id SERIAL PRIMARY KEY,
            user_id VARCHAR(255) NOT NULL,
            context JSONB NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("ошибка выполнения запроса: %v", err)
		}
	}

	return nil
}

func initializeAuthService() (*grpc.ClientConn, error) {
	authConfig, err := auth.NewConfig(".auth.env")
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации авторизации: %v", err)
	}

	conn, err := grpc.Dial(authConfig.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к сервису авторизации: %v", err)
	}

	auth.InitClient(conn)
	return conn, nil
}

func setupRouter() *chi.Mux {
	router := chi.NewRouter()

	// CORS middleware
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Основные middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.CleanPath)
	router.Use(middleware.StripSlashes)

	return router
}

func registerRoutes(r *chi.Mux, chatHandler *handler.Handler, projectAssistant *projectAI.ProjectAssistantHandler) {
	r.Route("/v1", func(r chi.Router) {
		// Middleware для проверки Content-Type
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		// Основные эндпоинты чата
		r.Post("/chat", chatHandler.Chat)
		r.Post("/clear-history", chatHandler.ClearHistory)

		// Эндпоинты AI-ассистента проектов
		projectAssistant.RegisterRoutes(r)

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	})
}

func setupServer(router *chi.Mux) *http.Server {
	return &http.Server{
		Addr:         ":5641",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func startServer(srv *http.Server) {
	log.Printf("Сервер запущен на порту %s\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка запуска сервера: %s\n", err)
	}
}

func waitForShutdown(srv *http.Server) {
	// Канал для получения сигналов операционной системы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ожидание сигнала
	<-quit
	log.Println("Получен сигнал завершения, начинаем graceful shutdown...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Пытаемся gracefully остановить сервер
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Ошибка при остановке сервера:", err)
	}

	log.Println("Сервер успешно остановлен")
}
