package database

import (
	"fmt"
	"log"
	"strconv"

	"github.com/azevedoguigo/demostore_api.git/internal/config"
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := "host=" + cfg.Postgres.DBHost + " user=" + cfg.Postgres.DBUser + " password=" + cfg.Postgres.DBPassword +
		" dbname=" + cfg.Postgres.DBName + " port=" + strconv.Itoa(cfg.Postgres.DBPort) + " sslmode=" + cfg.Postgres.DBSSLMode

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&domain.User{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database: %w", err)
	}

	log.Println("Database connection established successfully")
	return db, nil
}
