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

	"github.com/0Bleak/clayjar-jar-service/internal/config"
	"github.com/0Bleak/clayjar-jar-service/internal/discovery"
	"github.com/0Bleak/clayjar-jar-service/internal/handlers"
	"github.com/0Bleak/clayjar-jar-service/internal/messaging"
	"github.com/0Bleak/clayjar-jar-service/internal/repository"
	"github.com/0Bleak/clayjar-jar-service/internal/service"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	log.Println("Connected to MongoDB")

	db := mongoClient.Database(cfg.MongoDB)
	jarRepo := repository.NewJarRepository(db)

	if err := jarRepo.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	// Initialize Kafka Producer
	kafkaProducer := messaging.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer kafkaProducer.Close()
	log.Println("Kafka producer initialized")

	// Initialize Service and Handler
	jarService := service.NewJarService(jarRepo, kafkaProducer)
	jarHandler := handlers.NewJarHandler(jarService)

	// Setup Router
	router := mux.NewRouter()
	jarHandler.RegisterRoutes(router)

	// Register with Consul
	consulClient, err := discovery.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		return fmt.Errorf("failed to create consul client: %w", err)
	}

	serviceID := fmt.Sprintf("jar-service-%s", cfg.ServiceID)
	if err := consulClient.RegisterService(serviceID, "jar-service", cfg.ServerPort); err != nil {
		return fmt.Errorf("failed to register service with consul: %w", err)
	}
	log.Printf("Registered with Consul as %s", serviceID)

	defer func() {
		if err := consulClient.DeregisterService(serviceID); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		}
	}()

	// Start HTTP Server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting jar-service on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited gracefully")
	return nil
}
