package storage

import (
	"fmt"
	"log"

	"github.com/Shishlyannikovvv/project-avito/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(host, user, password, dbname, port string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Автомиграция - создает таблицы на основе структур из domain/models.go
	err = db.AutoMigrate(&domain.Team{}, &domain.User{}, &domain.PullRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Connected to PostgreSQL and ran migrations successfully")
	return db, nil
}
