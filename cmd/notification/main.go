package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/smukkama/weather-server/internal/notification"
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

	fmt.Println("Starting Notification Service...")

	// Create email notifier
	notifier := notification.NewEmailNotifier(&cfg.SMTP)

	// Test SMTP connection (optional, will skip if not configured)
	if err := notifier.TestConnection(); err != nil {
		fmt.Printf("Note: %v (notifications will be logged only)\n", err)
	}

	// Create consumer for alarm notifications
	consumer := queue.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicAlarms, "notification-group")
	defer consumer.Close()
	fmt.Println("Kafka consumer initialized")

	ctx := context.Background()

	fmt.Println("\n✓ Notification Service is running")
	fmt.Println("✓ Press Ctrl+C to stop")

	// Start consuming notifications
	go func() {
		for {
			msg, err := consumer.Consume(ctx)
			if err != nil {
				log.Printf("Failed to consume message: %v\n", err)
				continue
			}

			// Decode alarm notification
			alarmNotification, err := protocol.DecodeAlarmNotification(msg.Value)
			if err != nil {
				log.Printf("Failed to decode notification: %v\n", err)
				consumer.Commit(ctx, msg)
				continue
			}

			// Send notification
			if err := notifier.SendAlarmNotification(alarmNotification); err != nil {
				log.Printf("Failed to send notification: %v\n", err)
				// Don't commit on error - retry
				continue
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
