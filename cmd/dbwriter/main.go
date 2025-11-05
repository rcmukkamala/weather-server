package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/queue"
	"github.com/smukkama/weather-server/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Starting Database Writer Service...")
	db, err := database.Connect(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Connected to database")

	if err := db.RunMigrations("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Kafka consumer
	consumer := queue.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicMetrics, "dbwriter-group")
	defer consumer.Close()
	fmt.Println("Kafka consumer created (registering with broker...)")

	// Create batch writer (batch size: 100, flush interval: 5 seconds)
	batchWriter := queue.NewBatchWriter(consumer, db, 100, 5*time.Second)
	ctx := context.Background()
	// Start batch writer
	if err := batchWriter.Start(ctx); err != nil {
		log.Fatalf("Failed to start batch writer: %v", err)
	}
	fmt.Println("Batch writer started")

	// Print consumer stats periodically
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := consumer.Stats()
			fmt.Printf("Consumer stats: Messages=%d, Bytes=%d, Errors=%d\n",
				stats.Messages, stats.Bytes, stats.Errors)
		}
	}()

	fmt.Println("\n✓ Database Writer Service is running")
	fmt.Println("✓ Consuming from Kafka and writing to PostgreSQL")
	fmt.Println("✓ Batch size: 100 messages | Flush interval: 5 seconds")
	fmt.Println("✓ Consumer group will register when first message is consumed")
	fmt.Println("✓ Press Ctrl+C to stop")
	fmt.Println("\nWaiting for messages...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
	batchWriter.Stop()
	fmt.Println("Database Writer Service stopped")
}
