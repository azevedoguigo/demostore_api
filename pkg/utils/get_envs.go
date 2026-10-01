package utils

import (
	"log"
	"os"
	"strconv"
	"time"
)

func GetEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func GetEnvAsInt(key string, defaultValue int) int {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Error converting environment variable %s to int: %v", key, err)
		return defaultValue
	}
	return value
}

// GetEnvAsDuration parses values such as "30m" or "2h", falling back to defaultValue when unset or invalid.
func GetEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil || value <= 0 {
		log.Printf("Error converting environment variable %s to duration: %q", key, valueStr)
		return defaultValue
	}
	return value
}
