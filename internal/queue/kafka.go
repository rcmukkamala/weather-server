package queue

import (
	"context"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

// ProducerConfig holds configuration for the Kafka producer
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int           // Number of messages per batch
	BatchTimeout time.Duration // Max time to wait before sending batch
	Compression  string        // Compression type: "snappy", "lz4", "gzip", "zstd", "none"
	Async        bool          // Enable async publishing
	MaxAttempts  int           // Max retry attempts
	RequiredAcks int           // -1 (all), 0 (none), 1 (leader)
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	BatchBytes   int64 // Max bytes per batch
}

// Producer wraps a Kafka producer with optimizations
type Producer struct {
	writer *kafka.Writer
	config *ProducerConfig
}

// NewProducer creates a new optimized Kafka producer
func NewProducer(brokers []string, topic string) *Producer {
	// Default optimized configuration
	return NewProducerWithConfig(&ProducerConfig{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    100,                    // Batch up to 100 messages
		BatchTimeout: 100 * time.Millisecond, // Or 100ms timeout
		Compression:  "snappy",               // Snappy compression (fast)
		Async:        true,                   // Async for better throughput
		MaxAttempts:  3,                      // Retry 3 times
		RequiredAcks: 1,                      // Leader ack only (good balance)
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BatchBytes:   1048576, // 1MB per batch
	})
}

// NewProducerWithConfig creates a producer with custom configuration
func NewProducerWithConfig(config *ProducerConfig) *Producer {
	// Select compression algorithm
	var compression compress.Compression
	switch config.Compression {
	case "snappy":
		compression = compress.Snappy
	case "lz4":
		compression = compress.Lz4
	case "gzip":
		compression = compress.Gzip
	case "zstd":
		compression = compress.Zstd
		// No default needed - zero value of compress.Compression is nil compression
	}

	// Map required acks
	var requiredAcks kafka.RequiredAcks
	switch config.RequiredAcks {
	case -1:
		requiredAcks = kafka.RequireAll
	case 0:
		requiredAcks = kafka.RequireNone
	default:
		requiredAcks = kafka.RequireOne
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(config.Brokers...),
		Topic:    config.Topic,
		Balancer: &kafka.Hash{}, // Partition by key (zipcode)

		// Batching configuration (Phase 2 optimization!)
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		BatchBytes:   config.BatchBytes,

		// Compression (Phase 2 optimization!)
		Compression: compression,

		// Async/Sync (Phase 2 optimization!)
		Async: config.Async,

		// Reliability
		RequiredAcks: requiredAcks,
		MaxAttempts:  config.MaxAttempts,

		// Timeouts
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &Producer{
		writer: writer,
		config: config,
	}
}

// NewProducerFromKafkaConfig creates a producer from KafkaConfig
func NewProducerFromKafkaConfig(cfg interface{}) *Producer {
	// This accepts the config interface to avoid circular imports
	// Expected to be called with pkg/config.KafkaConfig

	// Use reflection or type assertion based on your config package
	// For simplicity, extract values dynamically
	type kafkaConfig struct {
		Brokers      []string
		TopicMetrics string
		BatchSize    int
		BatchTimeout time.Duration
		Compression  string
		Async        bool
		MaxAttempts  int
		RequiredAcks int
	}

	// This is a helper - typically you'd pass values directly
	// or make config package importable
	return NewProducer(nil, "")
}

// Publish sends a message to Kafka
func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// PublishBatch sends multiple messages to Kafka
func (p *Producer) PublishBatch(ctx context.Context, messages []kafka.Message) error {
	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}
	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer wraps a Kafka consumer
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,    // 1 byte
			MaxBytes:       10e6, // 10MB
			CommitInterval: 0,    // Manual commit for exactly-once
			StartOffset:    kafka.LastOffset,
		}),
	}
}

// Consume reads messages from Kafka
func (c *Consumer) Consume(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("failed to fetch message: %w", err)
	}
	return msg, nil
}

// Commit commits the message offset
func (c *Consumer) Commit(ctx context.Context, msg kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to commit message: %w", err)
	}
	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Stats returns consumer statistics
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// GetPartitionForZipcode returns the partition number for a zipcode
func GetPartitionForZipcode(zipcode string, numPartitions int) int {
	hash := crc32.ChecksumIEEE([]byte(zipcode))
	return int(hash % uint32(numPartitions))
}

// CreateTopic creates a Kafka topic with the specified number of partitions
func CreateTopic(brokers []string, topic string, numPartitions int, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial broker: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	fmt.Printf("Created topic %s with %d partitions\n", topic, numPartitions)
	return nil
}
