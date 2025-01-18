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

  "github.com/your_username/mistral/internal/auth"
  "github.com/your_username/mistral/internal/config"
  "github.com/your_username/mistral/internal/handler"
  "github.com/your_username/mistral/internal/repository"

  "github.com/go-chi/chi/v5/middleware"
  "github.com/go-chi/cors"
  "github.com/jmoiron/sqlx"
  _ "github.com/lib/pq"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.NewConfig(".env")
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к базе данных
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PgHost, cfg.PgPort, cfg.PgUser, cfg.PgPassword, cfg.PgName)

	db, err := sqlx.Connect("postgres", dbURL)
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

	auth.InitClient(conn)

	// Инициализация репозитория и хендлера
	repo := repository.NewRepository(db)
	handler := handler.NewHandler(repo, cfg.MistralApiKey, cfg.ModelName)

	// Настройка роутера
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Post("/v1/chat", handler.Chat)

	// Настройка и запуск сервера
	srv := &http.Server{
		Addr:    ":5641",
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
