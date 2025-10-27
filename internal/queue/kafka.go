package queue

import (
	"context"
	"fmt"
	"hash/crc32"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a Kafka producer
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{}, // Partition by key (zipcode)
			RequiredAcks: kafka.RequireOne,
			Async:        false, // Synchronous for reliability
		},
	}
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
