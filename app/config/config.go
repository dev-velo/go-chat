package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{
		Port: envOrDefault("PORT", "8080"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
