package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/smukkama/weather-server/internal/alarming"
	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/protocol"
	"github.com/smukkama/weather-server/internal/queue"
	"github.com/smukkama/weather-server/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Starting Alarming Service...")

	// Connect to database
	db, err := database.Connect(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Connected to database")

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis")

	// Create state manager
	stateManager := alarming.NewStateManager(redisClient)

	// Create alarm producer (for notifications)
	alarmProducer := queue.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicAlarms)
	defer alarmProducer.Close()
	fmt.Println("Alarm notification producer initialized")

	// Create evaluator
	evaluator := alarming.NewEvaluator(db, stateManager, alarmProducer)

	// Create consumer for metrics
	consumer := queue.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicMetrics, "alarming-group")
	defer consumer.Close()
	fmt.Println("Kafka consumer initialized")

	fmt.Println("\n✓ Alarming Service is running")
	fmt.Println("✓ Press Ctrl+C to stop")

	// Start consuming and evaluating
	go func() {
		for {
			msg, err := consumer.Consume(ctx)
			if err != nil {
				log.Printf("Failed to consume message: %v\n", err)
				continue
			}

			// Decode metric message
			metricMsg, err := protocol.DecodeMetricMessage(msg.Value)
			if err != nil {
				log.Printf("Failed to decode message: %v\n", err)
				consumer.Commit(ctx, msg)
				continue
			}

			// Evaluate metric
			if err := evaluator.EvaluateMetric(ctx, metricMsg); err != nil {
				log.Printf("Failed to evaluate metric: %v\n", err)
			}

			// Commit offset
			if err := consumer.Commit(ctx, msg); err != nil {
				log.Printf("Failed to commit offset: %v\n", err)
			}
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
}
