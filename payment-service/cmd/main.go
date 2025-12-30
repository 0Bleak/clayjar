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

	"github.com/0Bleak/payment-service/internal/config"
	"github.com/0Bleak/payment-service/internal/discovery"
	"github.com/0Bleak/payment-service/internal/handlers"
	"github.com/0Bleak/payment-service/internal/messaging"
	"github.com/0Bleak/payment-service/internal/repository"
	"github.com/0Bleak/payment-service/internal/service"
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
	kafkaConsumer := messaging.NewKafkaConsumer(cfg.KafkaBrokers, "order-events", "payment-service-group")
	defer kafkaConsumer.Close()
	log.Println("Kafka consumer initialized")

	// Initialize repositories and services
	paymentRepo := repository.NewPaymentRepository(db)
	paymentService := service.NewPaymentService(paymentRepo, kafkaProducer)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	// Start consuming order events
	go func() {
		if err := kafkaConsumer.ConsumeOrderEvents(context.Background(), paymentService); err != nil {
			log.Printf("Error consuming order events: %v", err)
		}
	}()

	// Setup router
	router := mux.NewRouter()
	paymentHandler.RegisterRoutes(router)

	// Register with Consul
	consulClient, err := discovery.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		return fmt.Errorf("failed to create consul client: %w", err)
	}

	serviceID := fmt.Sprintf("payment-service-%s", cfg.ServiceID)
	if err := consulClient.RegisterService(serviceID, "payment-service", cfg.ServerPort); err != nil {
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
		log.Printf("Starting payment-service on port %s", cfg.ServerPort)
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
	CREATE TABLE IF NOT EXISTS payments (
		id SERIAL PRIMARY KEY,
		order_id INTEGER NOT NULL,
		amount DECIMAL(10, 2) NOT NULL,
		status VARCHAR(50) DEFAULT 'pending',
		payment_method VARCHAR(50) DEFAULT 'credit_card',
		transaction_id VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
	CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
	`

	_, err := db.Exec(schema)
	return err
}
