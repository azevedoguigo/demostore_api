package config

import (
	"log"
	"time"

	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/joho/godotenv"
)

type PostgresConfig struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	ServerPort string
}

type StripeConfig struct {
	SecretKey              string
	WebhookSecret          string
	BoletoExpiresAfterDays int
}

type OrderConfig struct {
	// PendingTTL is how long a new order may stay unpaid before it is cancelled and its stock released.
	PendingTTL time.Duration
}

type Config struct {
	Postgres PostgresConfig
	Stripe   StripeConfig
	Order    OrderConfig
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	return &Config{
		Postgres: PostgresConfig{
			DBHost:     utils.GetEnv("DB_HOST", "localhost"),
			DBPort:     utils.GetEnvAsInt("DB_PORT", 5432),
			DBUser:     utils.GetEnv("DB_USER", "postgres"),
			DBPassword: utils.GetEnv("DB_PASSWORD", "postgres"),
			DBName:     utils.GetEnv("DB_NAME", "postgres"),
			DBSSLMode:  utils.GetEnv("DB_SSL_MODE", "disable"),
			ServerPort: utils.GetEnv("SERVER_PORT", "8080"),
		},
		Stripe: StripeConfig{
			SecretKey:     utils.GetEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: utils.GetEnv("STRIPE_WEBHOOK_SECRET", ""),
			// Stripe accepts 0 to 60 days.
			BoletoExpiresAfterDays: min(max(utils.GetEnvAsInt("BOLETO_EXPIRES_AFTER_DAYS", 3), 0), 60),
		},
		Order: OrderConfig{
			PendingTTL: utils.GetEnvAsDuration("PENDING_ORDER_TTL", 30*time.Minute),
		},
	}
}
