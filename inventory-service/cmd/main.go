package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0Bleak/inventory-service/internal/config"
	"github.com/0Bleak/inventory-service/internal/discovery"
	"github.com/0Bleak/inventory-service/internal/handlers"
	"github.com/0Bleak/inventory-service/internal/messaging"
	"github.com/0Bleak/inventory-service/internal/repository"
	"github.com/0Bleak/inventory-service/internal/service"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to PostgreSQL
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("Connected to PostgreSQL")

	// Run migrations
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize Kafka producer
	kafkaProducer := messaging.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer kafkaProducer.Close()
	log.Println("Kafka producer initialized")

	// Initialize Kafka consumer for order events
	kafkaConsumer := messaging.NewKafkaConsumer(cfg.KafkaBrokers, "order-events", "inventory-service-group")
	defer kafkaConsumer.Close()
	log.Println("Kafka consumer initialized")

	// Initialize repositories and services
	inventoryRepo := repository.NewInventoryRepository(db)
	inventoryService := service.NewInventoryService(inventoryRepo, kafkaProducer)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

	// Start consuming order events
	go func() {
		if err := kafkaConsumer.ConsumeOrderEvents(context.Background(), inventoryService); err != nil {
			log.Printf("Error consuming order events: %v", err)
		}
	}()

	// Setup router
	router := mux.NewRouter()
	inventoryHandler.RegisterRoutes(router)

	// Register with Consul
	consulClient, err := discovery.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		return fmt.Errorf("failed to create consul client: %w", err)
	}

	serviceID := fmt.Sprintf("inventory-service-%s", cfg.ServiceID)
	if err := consulClient.RegisterService(serviceID, "inventory-service", cfg.ServerPort); err != nil {
		return fmt.Errorf("failed to register service with consul: %w", err)
	}
	log.Printf("Registered with Consul as %s", serviceID)

	defer func() {
		if err := consulClient.DeregisterService(serviceID); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting inventory-service on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited gracefully")
	return nil
}

func runMigrations(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS inventory (
		id SERIAL PRIMARY KEY,
		jar_id VARCHAR(255) UNIQUE NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 0,
		reserved INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_inventory_jar_id ON inventory(jar_id);
	`

	_, err := db.Exec(schema)
	return err
}
