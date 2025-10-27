package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smukkama/weather-server/internal/aggregation"
	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/timer"
	"github.com/smukkama/weather-server/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Starting Aggregation Service...")

	// Connect to database
	db, err := database.Connect(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Connected to database")

	// Create timer manager
	timerManager := timer.NewTimerManager(2)
	timerManager.Start()
	defer timerManager.Stop()
	fmt.Println("Timer manager started")

	// Create aggregators
	hourlyAgg := aggregation.NewHourlyAggregator(db)
	dailyAgg := aggregation.NewDailyAggregator(db)

	// Schedule hourly aggregation
	scheduleHourlyAggregation(timerManager, hourlyAgg, cfg.Aggregation.HourlyDelay)

	// Schedule daily aggregation
	scheduleDailyAggregation(timerManager, dailyAgg, cfg.Aggregation.DailyTime)

	fmt.Println("\n✓ Aggregation Service is running")
	fmt.Println("✓ Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
}

func scheduleHourlyAggregation(tm *timer.TimerManager, agg *aggregation.HourlyAggregator, delay time.Duration) {
	taskID := "hourly-aggregation"

	var scheduleNext func()
	scheduleNext = func() {
		nextRun := agg.CalculateNextRunTime(delay)
		fmt.Printf("Next hourly aggregation scheduled for: %s\n", nextRun.Format("2006-01-02 15:04:05"))

		callback := func() {
			fmt.Println("\n--- Running Hourly Aggregation ---")
			if err := agg.AggregatePreviousHour(); err != nil {
				log.Printf("Hourly aggregation failed: %v\n", err)
			}
			fmt.Println("--- Hourly Aggregation Complete ---")

			// Schedule next run
			scheduleNext()
		}

		tm.Schedule(taskID, nextRun, callback)
	}

	scheduleNext()
}

func scheduleDailyAggregation(tm *timer.TimerManager, agg *aggregation.DailyAggregator, timeOfDay string) {
	taskID := "daily-aggregation"

	var scheduleNext func()
	scheduleNext = func() {
		nextRun, err := agg.CalculateNextRunTime(timeOfDay)
		if err != nil {
			log.Fatalf("Failed to calculate daily run time: %v", err)
		}
		fmt.Printf("Next daily aggregation scheduled for: %s\n", nextRun.Format("2006-01-02 15:04:05"))

		callback := func() {
			fmt.Println("\n--- Running Daily Aggregation ---")
			if err := agg.AggregatePreviousDay(); err != nil {
				log.Printf("Daily aggregation failed: %v\n", err)
			}
			fmt.Println("--- Daily Aggregation Complete ---")

			// Schedule next run
			scheduleNext()
		}

		tm.Schedule(taskID, nextRun, callback)
	}

	scheduleNext()
}
