package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"appstore/backend/internal/api"
	"appstore/backend/internal/catalog"
	"appstore/backend/internal/k8s"
	"appstore/backend/internal/rabbitmq"
)

func main() {
	var (
		addr        string
		rabbitmqURL string
		kubeconfig  string
		catalogPath string
	)

	flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
	flag.StringVar(&rabbitmqURL, "rabbitmq-url", "amqp://appstore:appstore@localhost:5672/appstore",
		"RabbitMQ connection URL")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (uses in-cluster config if empty)")
	flag.StringVar(&catalogPath, "catalog-path", "charts/catalog.yaml", "Path to catalog.yaml file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting appstore backend", "addr", addr)

	// Initialize catalog service
	catalogService := catalog.NewService(catalogPath)
	if err := catalogService.Load(); err != nil {
		logger.Error("Failed to load catalog", "error", err, "path", catalogPath)
		os.Exit(1)
	}
	logger.Info("Catalog loaded", "path", catalogPath, "apps", len(catalogService.ListApps()))

	// Initialize Kubernetes client (optional - deployment endpoints won't work without it)
	var k8sClient *k8s.Client
	k8sClient, err := k8s.NewClient(kubeconfig)
	if err != nil {
		logger.Warn("Failed to create Kubernetes client - deployment endpoints will be unavailable", "error", err)
	} else {
		logger.Info("Kubernetes client initialized")
	}

	// Initialize RabbitMQ publisher (optional - create deployment won't work without it)
	var publisher *rabbitmq.Publisher
	publisher = rabbitmq.NewPublisher(rabbitmq.PublisherConfig{
		URL:      rabbitmqURL,
		Exchange: "appstore",
	})

	if err := publisher.Connect(); err != nil {
		logger.Warn("Failed to connect to RabbitMQ - create deployment will be unavailable", "error", err)
		publisher = nil
	} else {
		defer publisher.Close()
		logger.Info("Connected to RabbitMQ", "url", rabbitmqURL)
	}

	// Initialize router
	router := api.NewRouter(publisher, k8sClient, catalogService)

	// Create HTTP server
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server stopped")
}
