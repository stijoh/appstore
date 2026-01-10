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
	"appstore/backend/internal/rabbitmq"
)

func main() {
	var (
		addr        string
		rabbitmqURL string
	)

	flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
	flag.StringVar(&rabbitmqURL, "rabbitmq-url", "amqp://appstore:appstore@localhost:5672/appstore",
		"RabbitMQ connection URL")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting appstore backend", "addr", addr)

	// Initialize RabbitMQ publisher
	publisher := rabbitmq.NewPublisher(rabbitmq.PublisherConfig{
		URL:      rabbitmqURL,
		Exchange: "appstore",
	})

	if err := publisher.Connect(); err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer publisher.Close()

	logger.Info("Connected to RabbitMQ", "url", rabbitmqURL)

	// Initialize router
	router := api.NewRouter(publisher)

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
