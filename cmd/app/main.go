package main

import (
	"context"
	"log"
	"os"

	"github.com/Shishlyannikovvv/project-avito/internal/api"
	"github.com/Shishlyannikovvv/project-avito/internal/service"
	"github.com/Shishlyannikovvv/project-avito/internal/storage"
)

func main() {
	// --- Конфигурация из переменных окружения (для Docker) ---
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	serverPort := "8080" // Требование ТЗ

	if dbHost == "" {
		// Заглушка для локального запуска без Docker-Compose, если нужно
		log.Fatal("DB_HOST environment variable not set. Please run via docker-compose.")
	}

	ctx := context.Background()

	// 1. Storage Layer (Подключение к БД)
	db, err := storage.NewPostgresDB(dbHost, dbUser, dbPassword, dbName, dbPort)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	repo := storage.NewRepository(db)

	// 2. Service Layer (Бизнес-логика)
	manager := service.NewManager(repo)

	// 3. API Layer (HTTP)
	handler := api.NewHandler(manager)
	router := api.SetupRouter(handler)

	// Запуск сервера
	log.Printf("Starting server on :%s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
