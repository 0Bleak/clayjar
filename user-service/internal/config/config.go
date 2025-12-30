package config

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

type Config struct {
	ServerPort  string
	ServiceID   string
	DatabaseURL string
	JWTSecret   string
	ConsulAddr  string
}

func LoadConfig() (*Config, error) {
	serviceID := os.Getenv("SERVICE_ID")
	if serviceID == "" {
		serviceID = uuid.New().String()
	}

	cfg := &Config{
		ServerPort:  getEnv("SERVER_PORT", "8081"),
		ServiceID:   serviceID,
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/clayjar?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		ConsulAddr:  getEnv("CONSUL_ADDR", "consul-server:8500"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
