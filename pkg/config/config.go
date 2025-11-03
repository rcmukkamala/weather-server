package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database    DatabaseConfig
	Redis       RedisConfig
	Kafka       KafkaConfig
	TCPServer   TCPServerConfig
	Aggregation AggregationConfig
	SMTP        SMTPConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers       []string
	TopicMetrics  string
	TopicAlarms   string
	NumPartitions int

	// Producer optimization settings
	BatchSize    int
	BatchTimeout time.Duration
	Compression  string
	Async        bool
	MaxAttempts  int
	RequiredAcks int
}

type TCPServerConfig struct {
	Port              int
	MaxConnections    int
	IdentifyTimeout   time.Duration
	InactivityTimeout time.Duration

	// Worker pool settings (Phase 1!)
	WorkerCount   int
	JobQueueSize  int
	UseWorkerPool bool
}

type AggregationConfig struct {
	HourlyDelay time.Duration
	DailyTime   string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not present)
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "weather_user"),
			Password: getEnv("DB_PASSWORD", "weather_pass"),
			DBName:   getEnv("DB_NAME", "weather_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			TopicMetrics:  getEnv("KAFKA_TOPIC_METRICS", "weather.metrics.raw"),
			TopicAlarms:   getEnv("KAFKA_TOPIC_ALARMS", "weather.alarms"),
			NumPartitions: getEnvAsInt("KAFKA_NUM_PARTITIONS", 10),

			// Producer optimization (Phase 2!)
			BatchSize:    getEnvAsInt("KAFKA_BATCH_SIZE", 100),
			BatchTimeout: getEnvAsDuration("KAFKA_BATCH_TIMEOUT", 100*time.Millisecond),
			Compression:  getEnv("KAFKA_COMPRESSION", "snappy"),
			Async:        getEnvAsBool("KAFKA_ASYNC", true),
			MaxAttempts:  getEnvAsInt("KAFKA_MAX_ATTEMPTS", 3),
			RequiredAcks: getEnvAsInt("KAFKA_REQUIRED_ACKS", 1),
		},
		TCPServer: TCPServerConfig{
			Port:              getEnvAsInt("TCP_PORT", 8080),
			MaxConnections:    getEnvAsInt("TCP_MAX_CONNECTIONS", 10000),
			IdentifyTimeout:   getEnvAsDuration("TCP_IDENTIFY_TIMEOUT", 10*time.Second),
			InactivityTimeout: getEnvAsDuration("TCP_INACTIVITY_TIMEOUT", 2*time.Minute),

			// Worker pool (Phase 1!) - default to 4x CPU cores
			WorkerCount:   getEnvAsInt("TCP_WORKER_COUNT", 0), // 0 = auto (4x cores)
			JobQueueSize:  getEnvAsInt("TCP_JOB_QUEUE_SIZE", 2000),
			UseWorkerPool: getEnvAsBool("TCP_USE_WORKER_POOL", true), // Enable by default
		},
		Aggregation: AggregationConfig{
			HourlyDelay: getEnvAsDuration("AGGREGATION_HOURLY_DELAY", 5*time.Minute),
			DailyTime:   getEnv("AGGREGATION_DAILY_TIME", "00:05"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnvAsInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "weather-server@example.com"),
			To:       getEnv("SMTP_TO", "admin@example.com"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
