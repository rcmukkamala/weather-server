package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/protocol"
)

// BatchWriter consumes from Kafka and batch-writes to database
type BatchWriter struct {
	consumer      *Consumer
	db            *database.DB
	batchSize     int
	flushInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewBatchWriter creates a new batch writer
func NewBatchWriter(consumer *Consumer, db *database.DB, batchSize int, flushInterval time.Duration) *BatchWriter {
	return &BatchWriter{
		consumer:      consumer,
		db:            db,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}
}

// Start begins consuming and writing to database
func (bw *BatchWriter) Start(ctx context.Context) error {
	bw.wg.Add(1)
	go bw.run(ctx)
	return nil
}

// Stop stops the batch writer gracefully
func (bw *BatchWriter) Stop() {
	close(bw.stopCh)
	bw.wg.Wait()
}

func (bw *BatchWriter) run(ctx context.Context) {
	defer bw.wg.Done()

	var batch []kafka.Message
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()

	// Consume messages in a goroutine (like your test program)
	msgChan := make(chan kafka.Message, 10)
	go func() {
		for {
			msg, err := bw.consumer.Consume(ctx)
			if err != nil {
				fmt.Printf("Consumer error: %v\n", err)
				continue
			}
			msgChan <- msg
		}
	}()

	for {
		select {
		case <-bw.stopCh:
			// Flush remaining batch before stopping
			if len(batch) > 0 {
				bw.flush(ctx, batch)
			}
			return

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				fmt.Printf("Flush interval reached (%d messages), flushing...\n", len(batch))
				bw.flush(ctx, batch)
				batch = nil
			}

		case msg := <-msgChan:
			fmt.Printf("Consumed message from topic (partition=%d, offset=%d)\n",
				msg.Partition, msg.Offset)
			batch = append(batch, msg)

			// Flush if batch is full
			if len(batch) >= bw.batchSize {
				fmt.Printf("Batch full (%d messages), flushing...\n", len(batch))
				bw.flush(ctx, batch)
				batch = nil
			}
		}
	}
}

func (bw *BatchWriter) flush(ctx context.Context, batch []kafka.Message) {
	if len(batch) == 0 {
		return
	}

	successCount := 0
	for _, msg := range batch {
		if err := bw.processMessage(msg); err != nil {
			fmt.Printf("Failed to process message: %v\n", err)
			continue
		}
		successCount++

		// Commit offset after successful processing
		if err := bw.consumer.Commit(ctx, msg); err != nil {
			fmt.Printf("Failed to commit offset: %v\n", err)
		}
	}

	fmt.Printf("Flushed batch of %d messages to database\n", successCount)
}

func (bw *BatchWriter) processMessage(msg kafka.Message) error {
	// Decode Kafka message
	metricMsg, err := protocol.DecodeMetricMessage(msg.Value)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Parse metric data
	parsedData, err := metricMsg.Data.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse metric data: %w", err)
	}

	// Ensure location exists
	location, err := bw.db.GetLocation(metricMsg.Zipcode)
	if err != nil {
		return fmt.Errorf("failed to get location: %w", err)
	}

	if location == nil {
		// Create location if it doesn't exist
		newLocation := &database.Location{
			Zipcode:  metricMsg.Zipcode,
			CityName: metricMsg.City,
		}
		if err := bw.db.UpsertLocation(newLocation); err != nil {
			return fmt.Errorf("failed to create location: %w", err)
		}
	}

	// Insert raw metric
	rawMetric := &database.RawMetric{
		Zipcode:        metricMsg.Zipcode,
		Timestamp:      parsedData.Timestamp,
		Temperature:    &parsedData.Temperature,
		Humidity:       &parsedData.Humidity,
		Precipitation:  &parsedData.Precipitation,
		WindSpeed:      &parsedData.WindSpeed,
		WindDirection:  &parsedData.WindDirection,
		PollutionIndex: &parsedData.PollutionIndex,
		PollenIndex:    &parsedData.PollenIndex,
		ReceivedAt:     metricMsg.ReceivedAt,
	}

	if err := bw.db.InsertRawMetric(rawMetric); err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}

	return nil
}
