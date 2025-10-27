package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smukkama/weather-server/internal/connection"
	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/queue"
	"github.com/smukkama/weather-server/internal/server"
	"github.com/smukkama/weather-server/internal/timer"
	"github.com/smukkama/weather-server/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Starting Weather Server...")

	// Connect to database
	db, err := database.Connect(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Connected to database")

	// Run migrations
	if err := db.RunMigrations("migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Kafka topics
	if err := queue.CreateTopic(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicMetrics,
		cfg.Kafka.NumPartitions,
		1, // replication factor
	); err != nil {
		fmt.Printf("Note: Topic creation failed (may already exist): %v\n", err)
	}

	if err := queue.CreateTopic(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicAlarms,
		1, // single partition for alarms
		1, // replication factor
	); err != nil {
		fmt.Printf("Note: Topic creation failed (may already exist): %v\n", err)
	}

	// Create Kafka producer
	producer := queue.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicMetrics)
	defer producer.Close()
	fmt.Println("Kafka producer initialized")

	// Create connection manager
	connManager := connection.NewManager(cfg.TCPServer.MaxConnections)
	fmt.Println("Connection manager initialized")

	// Create timer manager
	timerManager := timer.NewTimerManager(10) // 10 worker goroutines
	timerManager.Start()
	defer timerManager.Stop()
	fmt.Println("Timer manager started")

	// Create TCP server
	tcpServer := server.NewTCPServer(&cfg.TCPServer, connManager, timerManager, producer)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer tcpServer.Stop()

	// Start database writer
	consumer := queue.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicMetrics, "db-writer-group")
	defer consumer.Close()

	batchWriter := queue.NewBatchWriter(consumer, db, 100, 5*time.Second)
	if err := batchWriter.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start batch writer: %v", err)
	}
	defer batchWriter.Stop()
	fmt.Println("Database writer started")

	// Print statistics periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := connManager.Stats()
			timerStats := timerManager.Stats()
			fmt.Printf("\n--- Server Statistics ---\n")
			fmt.Printf("Active Connections: %d / %d\n", stats.TotalConnections, stats.MaxConnections)
			fmt.Printf("Unique Zipcodes: %d\n", stats.UniqueZipcodes)
			fmt.Printf("Scheduled Timers: %d\n", timerStats.ScheduledTasks)
			fmt.Printf("------------------------\n\n")
		}
	}()

	fmt.Println("\n✓ Weather Server is running")
	fmt.Printf("✓ TCP Server listening on port %d\n", cfg.TCPServer.Port)
	fmt.Println("✓ Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
}
