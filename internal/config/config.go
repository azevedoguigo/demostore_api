package config

import (
	"log"

	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	ServerPort string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	return &Config{
		DBHost:     utils.GetEnv("DB_HOST", "localhost"),
		DBPort:     utils.GetEnvAsInt("DB_PORT", 5432),
		DBUser:     utils.GetEnv("DB_USER", "postgres"),
		DBPassword: utils.GetEnv("DB_PASSWORD", "postgres"),
		DBName:     utils.GetEnv("DB_NAME", "postgres"),
		DBSSLMode:  utils.GetEnv("DB_SSL_MODE", "disable"),
		ServerPort: utils.GetEnv("SERVER_PORT", "8080"),
	}
}
