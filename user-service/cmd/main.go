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

	"github.com/0Bleak/user-service/internal/config"
	"github.com/0Bleak/user-service/internal/discovery"
	"github.com/0Bleak/user-service/internal/handlers"
	"github.com/0Bleak/user-service/internal/middleware"
	"github.com/0Bleak/user-service/internal/repository"
	"github.com/0Bleak/user-service/internal/service"
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

	// Initialize repositories and services
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, cfg.JWTSecret)
	userHandler := handlers.NewUserHandler(userService)

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/users/register", userHandler.Register).Methods(http.MethodPost)
	router.HandleFunc("/users/login", userHandler.Login).Methods(http.MethodPost)
	router.HandleFunc("/users/me", middleware.AuthMiddleware(userHandler.GetProfile, cfg.JWTSecret)).Methods(http.MethodGet)
	router.HandleFunc("/health", userHandler.HealthCheck).Methods(http.MethodGet)

	// Register with Consul
	consulClient, err := discovery.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		return fmt.Errorf("failed to create consul client: %w", err)
	}

	serviceID := fmt.Sprintf("user-service-%s", cfg.ServiceID)
	if err := consulClient.RegisterService(serviceID, "user-service", cfg.ServerPort); err != nil {
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
		log.Printf("Starting user-service on port %s", cfg.ServerPort)
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
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		full_name VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'customer',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	_, err := db.Exec(schema)
	return err
}
