# Weather Server

A production-grade TCP-based weather data collection and monitoring system built in Go, featuring real-time data ingestion, aggregation, and threshold-based alarming.

## 🌟 Features

- **TCP Server**: Multi-client TCP server with connection management
- **Real-time Data Ingestion**: 5-minute weather metric collection via Kafka
- **Custom Timer System**: Min-heap based timer for efficient connection timeouts
- **Automatic Aggregation**: Hourly averages and daily min/max calculations
- **Threshold Alarming**: Configurable alerts with duration-based triggers
- **Email Notifications**: SMTP-based alarm notifications
- **Scalable Architecture**: Kafka-based event streaming with consumer groups
- **State Management**: Redis-backed alarm state tracking

## 📋 Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Database Schema](#database-schema)
- [API Protocol](#api-protocol)
- [Services](#services)
- [Testing](#testing)
- [Monitoring](#monitoring)

## 🏗️ Architecture

```
┌──────────────┐
│ Weather      │
│ Clients      │ (TCP)
│ (Multiple)   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│          TCP Server (Port 8080)              │
│  - Connection Manager                        │
│  - Timer Manager (Min-Heap)                  │
│  - Protocol Handler                          │
└──────┬───────────────────────────────────────┘
       │
       ▼ (Kafka)
┌──────────────────────────────────────────────┐
│   weather.metrics.raw (10 partitions)        │
│   Partitioned by zipcode                     │
└──┬───────────────────────────────────────┬───┘
   │                                       │
   ▼                                       ▼
┌──────────────┐                  ┌──────────────┐
│ DB Writer    │                  │ Alarming     │
│ Service      │                  │ Service      │
│              │                  │              │
│ ↓ Batch      │                  │ ↓ Real-time  │
│ PostgreSQL   │                  │ Redis State  │
└──────────────┘                  └──────┬───────┘
                                         │
                                         ▼ (Kafka)
                                  ┌──────────────┐
                                  │ weather.     │
                                  │ alarms       │
                                  └──────┬───────┘
                                         │
                                         ▼
                                  ┌──────────────┐
                                  │ Notification │
                                  │ Service      │
                                  │              │
                                  │ ↓ SMTP       │
                                  └──────────────┘

┌──────────────┐
│ Aggregation  │
│ Service      │
│              │
│ - Hourly     │
│ - Daily      │
└──────────────┘
```

## 🔧 Prerequisites

- **Go**: 1.21 or later
- **Docker & Docker Compose**: For running dependencies
- **Make**: For build automation (optional)

## 🚀 Quick Start

### 1. Clone and Setup

```bash
cd Weather-Server
```

### 2. Start Infrastructure

```bash
# Start PostgreSQL, Redis, Kafka, Zookeeper
make docker-up

# Or manually:
docker-compose up -d
```

Wait ~10 seconds for services to be healthy.

### 3. Build Services

```bash
make build

# Or manually:
go build -o bin/server ./cmd/server
go build -o bin/aggregator ./cmd/aggregator
go build -o bin/alarming ./cmd/alarming
go build -o bin/notification ./cmd/notification
```

### 4. Start Services

Open 4 separate terminals:

```bash
# Terminal 1: TCP Server
make run-server

# Terminal 2: Aggregation Service
make run-aggregator

# Terminal 3: Alarming Service
make run-alarming

# Terminal 4: Notification Service
make run-notification
```

### 5. Test with Sample Client

```bash
# Run the sample weather client
go run examples/client/main.go
```

## ⚙️ Configuration

All configuration is via environment variables (`.env` file):

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=weather_user
DB_PASSWORD=weather_pass
DB_NAME=weather_db

# Redis
REDIS_ADDR=localhost:6379

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_METRICS=weather.metrics.raw
KAFKA_TOPIC_ALARMS=weather.alarms
KAFKA_NUM_PARTITIONS=10

# TCP Server
TCP_PORT=8080
TCP_MAX_CONNECTIONS=10000
TCP_IDENTIFY_TIMEOUT=10s
TCP_INACTIVITY_TIMEOUT=2m

# Aggregation
AGGREGATION_HOURLY_DELAY=5m       # Run at HH:05:00
AGGREGATION_DAILY_TIME=00:05      # Run at 00:05:00

# SMTP (optional - leave empty to skip email)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=weather-server@example.com
SMTP_TO=admin@example.com
```

## 🗄️ Database Schema

### Tables

**locations**
- Stores zipcode and city information

**raw_metrics**
- 5-minute weather measurements
- Indexed by (zipcode, timestamp)

**hourly_metrics**
- Hourly aggregated averages
- Calculated every hour at HH:05:00

**daily_summary**
- Daily min/max statistics
- Calculated daily at 00:05:00

**alarm_thresholds**
- Configurable alarm rules per zipcode/metric

**alarms_log**
- Historical log of triggered alarms

### Example: Add Alarm Threshold

```sql
-- Alert if wind speed > 74 mph for 15 minutes in Miami Beach
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('33139', 'wind_speed', '>', 74.0, 15, true);

-- Alert if temperature < -20°C for 60 minutes in Minneapolis
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('55401', 'temperature', '<', -20.0, 60, true);
```

## 📡 API Protocol

All messages are **newline-terminated JSON** over TCP.

### Client → Server

**1. Identify (on connect)**
```json
{"type": "identify", "zipcode": "90210", "city": "Beverly Hills"}
```

**2. Metrics (every 5 minutes)**
```json
{
  "type": "metrics",
  "data": {
    "timestamp": "2025-10-26T13:30:00Z",
    "temperature": 15.5,
    "humidity": 60.2,
    "precipitation": 0.0,
    "wind_speed": 12.0,
    "wind_direction": "NW",
    "pollution_index": 45.0,
    "pollen_index": 78.0
  }
}
```

**3. Keepalive (every 30-60s)**
```json
{"type": "keepalive"}
```

### Server → Client

**Acknowledgments**
```json
{"type": "ack", "status": "identified"}
{"type": "ack", "status": "alive"}
{"type": "ack", "status": "error"}
```

## 🔧 Services

### 1. TCP Server (`cmd/server`)

- Listens on port 8080
- Handles client connections
- Validates and forwards metrics to Kafka
- Custom min-heap timer for connection timeouts
- Automatic cleanup of inactive connections

### 2. Aggregation Service (`cmd/aggregator`)

- **Hourly**: Runs at HH:05:00, aggregates previous hour
- **Daily**: Runs at 00:05:00, aggregates previous day
- Uses custom timer manager for scheduling

### 3. Alarming Service (`cmd/alarming`)

- Consumes metrics in real-time from Kafka
- Evaluates against configured thresholds
- Manages alarm state machine in Redis
- States: CLEAR → PENDING_ALARM → ALARMING
- Publishes notifications to alarm topic

### 4. Notification Service (`cmd/notification`)

- Consumes alarm notifications from Kafka
- Sends email alerts via SMTP
- Handles both triggered and cleared alarms

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Test specific package
go test -v ./internal/timer/
go test -v ./internal/connection/
```

### Manual Testing

```bash
# Connect with netcat
nc localhost 8080

# Send identify message
{"type": "identify", "zipcode": "90210", "city": "Beverly Hills"}

# Send metrics
{"type": "metrics", "data": {"timestamp": "2025-10-26T13:30:00Z", "temperature": 25.5, "humidity": 60.2, "precipitation": 0.0, "wind_speed": 12.0, "wind_direction": "NW", "pollution_index": 45.0, "pollen_index": 78.0}}

# Send keepalive
{"type": "keepalive"}
```

## 📊 Monitoring

### Kafka UI

Access Kafka UI at: http://localhost:8090

Monitor:
- Topic partitions and lag
- Consumer groups
- Message throughput

### Redis CLI

```bash
docker exec -it weather-redis redis-cli

# View alarm states
KEYS alarm_state:*
GET alarm_state:90210:wind_speed
```

### PostgreSQL

```bash
docker exec -it weather-postgres psql -U weather_user -d weather_db

# View recent metrics
SELECT * FROM raw_metrics ORDER BY timestamp DESC LIMIT 10;

# View active alarms
SELECT * FROM alarms_log WHERE status = 'ACTIVE';

# View hourly aggregations
SELECT * FROM hourly_metrics ORDER BY hour_timestamp DESC LIMIT 24;
```

## 🎯 Key Design Decisions

### 1. Why Kafka over RabbitMQ?

- **Time-series nature**: Weather data is inherently time-ordered
- **Replay capability**: Critical for debugging alarm logic
- **Multiple consumers**: DB writer and alarming service read independently
- **Durability**: Message persistence with configurable retention
- **Partitioning**: Natural partitioning by zipcode for parallel processing

### 2. Custom Min-Heap Timer

- Requirement from design document
- Efficient O(log n) for scheduling 10,000+ connections
- Centralized timer management vs. 10,000 individual goroutines
- Better visibility and monitoring

### 3. Redis for Alarm State

- Fast in-memory access for state machine
- Persistence for reliability
- Automatic expiry for cleanup
- Distributed state management if scaled

## 📁 Project Structure

```
Weather-Server/
├── cmd/
│   ├── server/         # TCP server main
│   ├── aggregator/     # Aggregation service main
│   ├── alarming/       # Alarming service main
│   └── notification/   # Notification service main
├── internal/
│   ├── protocol/       # Message types and parsing
│   ├── connection/     # Connection manager
│   ├── timer/          # Custom min-heap timer
│   ├── queue/          # Kafka abstraction
│   ├── database/       # DB models and operations
│   ├── aggregation/    # Aggregation logic
│   ├── alarming/       # Alarm state machine
│   └── notification/   # Email notifications
├── pkg/
│   └── config/         # Configuration management
├── migrations/         # Database migrations
├── examples/
│   └── client/         # Sample weather client
├── docker-compose.yml  # Infrastructure
├── Makefile           # Build automation
└── README.md          # This file
```

## 🔐 Security Considerations

- Use strong passwords for PostgreSQL, Redis
- Enable TLS for Kafka in production
- Use app-specific passwords for SMTP
- Validate all client input
- Rate limiting for TCP connections
- SQL injection prevention (parameterized queries)

## 🚀 Production Deployment

### Scaling Strategies

1. **TCP Server**: Run multiple instances behind load balancer
2. **DB Writer**: Scale by increasing batch size or adding instances
3. **Alarming Service**: Scale by increasing Kafka partitions
4. **Aggregation**: Single instance sufficient (scheduled tasks)

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weather-tcp-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: weather-server
  template:
    metadata:
      labels:
        app: weather-server
    spec:
      containers:
      - name: server
        image: weather-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: KAFKA_BROKERS
          value: "kafka-service:9092"
```

## 📝 License

MIT License

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## 📧 Contact

For questions or issues, please open a GitHub issue.

---

**Built with ❤️ using Go, Kafka, PostgreSQL, and Redis**

