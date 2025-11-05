# Weather Server - Service Startup Guide

## Complete Architecture

```
┌──────────┐
│  Device  │
└────┬─────┘
     │ TCP Connection
     ▼
┌─────────────────┐
│   TCP Server    │ (cmd/server)
└────┬────────────┘
     │ Produces
     ▼
┌─────────────────┐
│  Kafka Topic    │ weather.metrics.raw
│  (Message Bus)  │
└─┬───────────┬───┘
  │           │
  │           ├──────────────────────────┐
  │           │                          │
  ▼           ▼                          ▼
┌───────────────┐  ┌─────────────┐  ┌─────────────┐
│  DB Writer    │  │  Alarming   │  │ (Future)    │
│  Service      │  │  Service    │  │ Analytics   │
└───────┬───────┘  └──────┬──────┘  └─────────────┘
        │                 │
        │ Writes          │ Produces Alarms
        ▼                 ▼
  ┌──────────┐      ┌─────────────┐
  │PostgreSQL│      │Kafka Topic  │ weather.alarms
  │ Database │      └──────┬──────┘
  └────┬─────┘             │
       │                   ▼
       │            ┌─────────────────┐
       │            │  Notification   │
       │            │    Service      │
       │            └─────────────────┘
       │
       ▼
  ┌──────────────┐
  │ Aggregator   │ (Scheduled: Hourly/Daily)
  │   Service    │
  └──────────────┘
```

## Service Roles

### 1. **TCP Server** (`cmd/server`)
- Listens on TCP port 8080
- Receives metrics from IoT devices
- Publishes to Kafka `weather.metrics.raw` topic
- **Must run first** (or data won't be collected)

### 2. **Database Writer** (`cmd/dbwriter`) ⭐ NEW
- Consumes from Kafka `weather.metrics.raw` topic
- Batch writes metrics to PostgreSQL
- Consumer group: `dbwriter-group`
- **Critical**: Without this, database stays empty!

### 3. **Alarming Service** (`cmd/alarming`)
- Consumes from Kafka `weather.metrics.raw` topic
- Evaluates alarm thresholds
- Publishes alarms to `weather.alarms` topic
- Consumer group: `alarming-group`

### 4. **Notification Service** (`cmd/notification`)
- Consumes from Kafka `weather.alarms` topic
- Sends email notifications
- Consumer group: `notification-group`

### 5. **Aggregator Service** (`cmd/aggregator`)
- Reads from PostgreSQL (not Kafka)
- Runs on schedule (hourly/daily)
- Creates aggregated statistics

## Starting Services

### Prerequisites
```bash
# Start infrastructure services
make docker-up

# Wait for Kafka to be healthy (~60 seconds)
docker inspect weather-kafka --format='{{.State.Health.Status}}'
# Should show: healthy
```

### Option A: Run All Services (Recommended Order)
```bash
# Terminal 1: TCP Server (receives data)
make run-server

# Terminal 2: Database Writer (stores to PostgreSQL) ⭐ IMPORTANT
make run-dbwriter

# Terminal 3: Alarming Service (monitors thresholds)
make run-alarming

# Terminal 4: Notification Service (sends alerts)
make run-notification

# Terminal 5: Aggregator (periodic aggregation)
make run-aggregator
```

### Option B: Minimal Setup (Server + DB Writer)
```bash
# Terminal 1: TCP Server
make run-server

# Terminal 2: Database Writer
make run-dbwriter

# This is enough for basic data collection!
```

## Verifying Services

### Check Kafka Consumers
```bash
# List all consumer groups
docker exec weather-kafka /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --list

# Should show:
# dbwriter-group      ⭐ NEW
# alarming-group
# notification-group
```

### Check Kafka UI
Open http://localhost:8090
- **Topics tab**: See `weather.metrics.raw` and `weather.alarms`
- **Consumers tab**: See all 3 consumer groups (dbwriter, alarming, notification)

### Check Database
```bash
# Connect to PostgreSQL
docker exec -it weather-postgres psql -U weather_user -d weather_db

# Check if data is being written
SELECT COUNT(*) FROM raw_metrics;
SELECT COUNT(*) FROM locations;
```

## Testing the System

### Send Test Data
```bash
# Send a test metric
echo '{"type":"identify","zipcode":"12345","city":"TestCity"}
{"type":"metrics","data":{"timestamp":"2025-11-04T10:00:00Z","temperature":22.5,"humidity":65,"precipitation":0,"wind_speed":10,"wind_direction":"N","pollution_index":45,"pollen_index":3}}' | nc localhost 8080
```

### Verify Data Flow
```bash
# 1. Check Kafka topic has messages
docker exec weather-kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic weather.metrics.raw \
  --from-beginning \
  --max-messages 1

# 2. Check database has the metric
docker exec -it weather-postgres psql -U weather_user -d weather_db \
  -c "SELECT * FROM raw_metrics ORDER BY received_at DESC LIMIT 1;"

# 3. Check dbwriter logs
# Should see: "Flushed batch of X messages to database"
```

## Common Issues

### Issue: "No consumers" in Kafka UI
**Cause**: Services using `localhost:9092` can't connect to Kafka  
**Solution**: Services are already configured correctly - just rebuild:
```bash
pkill -f "bin/"  # Stop all services
make build       # Rebuild with fixes
make run-dbwriter  # Restart
```

### Issue: Database is empty
**Cause**: Database Writer service not running  
**Solution**: 
```bash
make run-dbwriter  # This is the NEW required service!
```

### Issue: "EOF" errors in alarming/notification logs
**Cause**: Normal - happens when no messages available  
**Solution**: Send test data (see above) or ignore (it's expected)

### Issue: Permission denied when building
**Cause**: Old binaries still running  
**Solution**:
```bash
pkill -f "bin/"
make clean
make build
```

## Performance Configuration

The Database Writer uses batching for efficiency:

**Current defaults** (in `cmd/dbwriter/main.go`):
- Batch size: 100 messages
- Flush interval: 5 seconds

**To adjust for high throughput**:
```go
// Increase batch size for better throughput
batchWriter := queue.NewBatchWriter(consumer, db, 500, 10*time.Second)
```

**To adjust for low latency**:
```go
// Smaller batches, faster writes
batchWriter := queue.NewBatchWriter(consumer, db, 10, 1*time.Second)
```

## Service Dependencies

```
Infrastructure (Docker):
├── PostgreSQL (required by: dbwriter, aggregator, alarming)
├── Redis (required by: alarming)
└── Kafka (required by: server, dbwriter, alarming, notification)

Application Services:
├── server (no dependencies on other services)
├── dbwriter (depends on: server via Kafka)
├── alarming (depends on: server via Kafka)
├── notification (depends on: alarming via Kafka)
└── aggregator (depends on: dbwriter for database data)
```

## Next Steps

1. **Start infrastructure**: `make docker-up`
2. **Start server**: `make run-server`
3. **Start dbwriter**: `make run-dbwriter` ⭐ **CRITICAL**
4. **Send test data** (see Testing section)
5. **Verify in Kafka UI**: http://localhost:8090
6. **Check database**: See "Check Database" section above
7. **Start other services** as needed

---

**Key Takeaway**: The Database Writer service is now **required** for the system to work properly. It bridges Kafka and PostgreSQL!

