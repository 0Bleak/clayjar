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

	"github.com/0Bleak/api-gateway/internal/config"
	"github.com/0Bleak/api-gateway/internal/discovery"
	"github.com/0Bleak/api-gateway/internal/handlers"
	"github.com/0Bleak/api-gateway/internal/middleware"
	"github.com/0Bleak/api-gateway/internal/proxy"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

	consulClient, err := discovery.NewConsulClient(cfg.ConsulAddr)
	if err != nil {
		return fmt.Errorf("failed to create consul client: %w", err)
	}

	loadBalancer := proxy.NewLoadBalancer(consulClient)
	proxyHandler := handlers.NewProxyHandler(loadBalancer)

	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.RateLimitMiddleware(100))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)

	// Jar Service routes
	router.PathPrefix("/api/jars").Handler(proxyHandler.ProxyToService("jar-service"))

	// User Service routes
	router.PathPrefix("/api/users").Handler(proxyHandler.ProxyToService("user-service"))

	// Order Service routes
	router.PathPrefix("/api/orders").Handler(proxyHandler.ProxyToService("order-service"))

	// Inventory Service routes
	router.PathPrefix("/api/inventory").Handler(proxyHandler.ProxyToService("inventory-service"))

	// Payment Service routes
	router.PathPrefix("/api/payments").Handler(proxyHandler.ProxyToService("payment-service"))

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API Gateway starting on port %s", cfg.ServerPort)
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
