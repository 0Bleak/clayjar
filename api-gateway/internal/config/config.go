package config

import (
	"os"
)

type Config struct {
	ServerPort string
	ConsulAddr string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8000"),
		ConsulAddr: getEnv("CONSUL_ADDR", "localhost:8500"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
