package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

type Config struct {
	ServerPort   string
	ServiceID    string
	MongoURI     string
	MongoDB      string
	KafkaBrokers []string
	KafkaTopic   string
	ConsulAddr   string
}

func LoadConfig() (*Config, error) {
	serviceID := os.Getenv("SERVICE_ID")
	if serviceID == "" {
		serviceID = uuid.New().String()
	}

	cfg := &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		ServiceID:    serviceID,
		MongoURI:     getEnv("MONGO_URI", "mongodb://mongo:27017"),
		MongoDB:      getEnv("MONGO_DB", "clayjar"),
		KafkaBrokers: parseKafkaBrokers(getEnv("KAFKA_BROKERS", "shared-kafka:9092")),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "jar-events"),
		ConsulAddr:   getEnv("CONSUL_ADDR", "consul-server:8500"),
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
	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	if c.MongoDB == "" {
		return fmt.Errorf("MONGO_DB is required")
	}
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.KafkaTopic == "" {
		return fmt.Errorf("KAFKA_TOPIC is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseKafkaBrokers(brokers string) []string {
	return strings.Split(brokers, ",")
}
