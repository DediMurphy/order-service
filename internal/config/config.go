package config

import (
	"os"
	"time"
)

type Config struct {
	Port            string
	DBPath          string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:            getEnv("APP_PORT", "8080"),
		DBPath:          getEnv("DB_PATH", "order_service.db"),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
