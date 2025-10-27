# Weather Server - Quick Start Guide

Get the Weather Server running in 5 minutes on Ubuntu/Linux!

## Prerequisites

### Required Software

- Go 1.21+
- Docker & Docker Compose
- Make (build tool)
- Terminal (4 windows/tabs recommended)

### Ubuntu/Debian Installation

If you're on Ubuntu or Debian, install all prerequisites with:

```bash
# Update package list
sudo apt update

# Install Go 1.21+ (check for latest version)
sudo apt install golang-go -y

# Verify Go version (should be 1.21 or higher)
go version

# If Go version is too old, install from official source:
# wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
# sudo rm -rf /usr/local/go
# sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
# export PATH=$PATH:/usr/local/go/bin
# echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Install Docker
sudo apt install docker.io -y

# Install Docker Compose
sudo apt install docker-compose -y

# Add your user to docker group (avoid sudo for docker commands)
sudo usermod -aG docker $USER

# Apply group changes (or logout/login)
newgrp docker

# Install Make
sudo apt install make -y

# Verify installations
docker --version
docker-compose --version
make --version
```

### Other Linux Distributions

**Fedora/RHEL/CentOS:**
```bash
sudo dnf install golang docker docker-compose make -y
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
```

**Arch Linux:**
```bash
sudo pacman -S go docker docker-compose make
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
```

### Verify Prerequisites

```bash
# Check all tools are installed
go version       # Should show go1.21 or higher
docker --version # Should show Docker version
docker-compose --version # Should show docker-compose version
make --version   # Should show GNU Make version

# Test Docker (without sudo)
docker ps        # Should list containers (may be empty)
```

If `docker ps` requires sudo, you need to logout and login again for group changes to take effect.

## Step 1: Start Infrastructure (30 seconds)

```bash
cd Weather-Server
make docker-up
```

This starts:
- PostgreSQL (port 5432)
- Redis (port 6379)
- Kafka (port 9092)
- Zookeeper (port 2181)
- Kafka UI (port 8090)

Wait ~10 seconds for services to be healthy.

## Step 2: Build Services (15 seconds)

```bash
make build
```

Creates 4 binaries in `bin/`:
- `server` - TCP server
- `aggregator` - Hourly/daily aggregation
- `alarming` - Real-time alarm evaluation
- `notification` - Email notifications

## Step 3: Run Services (4 terminals)

### Terminal 1: TCP Server
```bash
make run-server
```
**Output:**
```
Starting Weather Server...
Connected to database
✓ Weather Server is running
✓ TCP Server listening on port 8080
```

### Terminal 2: Aggregation Service
```bash
make run-aggregator
```
**Output:**
```
Starting Aggregation Service...
Next hourly aggregation scheduled for: 2025-10-26 15:05:00
Next daily aggregation scheduled for: 2025-10-27 00:05:00
✓ Aggregation Service is running
```

### Terminal 3: Alarming Service
```bash
make run-alarming
```
**Output:**
```
Starting Alarming Service...
Connected to Redis
✓ Alarming Service is running
```

### Terminal 4: Notification Service
```bash
make run-notification
```
**Output:**
```
Starting Notification Service...
Note: SMTP not configured (notifications will be logged only)
✓ Notification Service is running
```

## Step 4: Test with Sample Client (5 seconds)

In a 5th terminal:

```bash
go run examples/client/main.go
```

**Output:**
```
Weather Client Starting...
Location: Beverly Hills, 90210
Server: localhost:8080

✓ Connected to server
→ Sent identify message
← Received ack: identified

✓ Client running (Ctrl+C to stop)

→ Sent metrics: temp=25.3°C, humidity=62.5%, wind=15.2 mph NW
→ Sent keepalive
← Received ack: alive
```

## ✅ Success!

You should see:
1. **Server Terminal**: Connection logs and metric receipts
2. **Client Terminal**: Periodic metrics and keepalives
3. **Alarming Terminal**: Evaluation messages (if thresholds exist)
4. **Kafka UI** (http://localhost:8090): Messages flowing

## Next Steps

### Configure Alarms

```bash
# Connect to PostgreSQL
docker exec -it weather-postgres psql -U weather_user -d weather_db

# Add alarm threshold
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('90210', 'temperature', '>', 30.0, 10, true);
```

### Configure Email Notifications

Edit `.env`:
```bash
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_TO=admin@example.com
```

Restart notification service to pick up changes.

### Monitor with Kafka UI

Open http://localhost:8090 in browser:
- View topics: `weather.metrics.raw`, `weather.alarms`
- Monitor consumer lag
- Inspect messages

### Check Database

```bash
docker exec -it weather-postgres psql -U weather_user -d weather_db

# View recent metrics
SELECT * FROM raw_metrics ORDER BY timestamp DESC LIMIT 10;

# View locations
SELECT * FROM locations;

# View alarm thresholds
SELECT * FROM alarm_thresholds WHERE is_active = true;
```

### Check Redis Alarm States

```bash
docker exec -it weather-redis redis-cli

# List all alarm states
KEYS alarm_state:*

# View specific state
GET alarm_state:90210:temperature
```

## Common Commands

```bash
# Stop all services
docker-compose down

# View Docker logs
docker-compose logs -f kafka
docker-compose logs -f postgres

# Rebuild after code changes
make build

# Run tests
make test

# Clean build artifacts
make clean
```

## Troubleshooting

### "Connection refused" to Kafka
- Wait 15-20 seconds after `docker-compose up`
- Check: `docker ps` - all containers should be "Up"
- Check logs: `docker-compose logs kafka`

### "Connection refused" to PostgreSQL
- Ensure migrations ran: Check server startup logs
- Manually run: `docker exec -it weather-postgres psql -U weather_user -d weather_db`

### No metrics in database
- Check server received them: Look for "Received metrics from..." logs
- Check Kafka UI: Are messages in `weather.metrics.raw`?
- Check DB writer logs in Terminal 1

### Alarms not triggering
- Verify threshold configured: `SELECT * FROM alarm_thresholds;`
- Check metric value exceeds threshold
- Ensure duration_minutes has elapsed (e.g., 10 minutes)
- Check alarming service logs

## Architecture Recap

```
Weather Client (examples/client/main.go)
    ↓ TCP (port 8080)
TCP Server (bin/server)
    ↓ Kafka
[weather.metrics.raw topic]
    ↓                      ↓
DB Writer              Alarming Service
    ↓                      ↓ Redis
PostgreSQL          [weather.alarms topic]
    ↓                      ↓
Aggregation         Notification Service
                           ↓ SMTP
```

## Performance Tips

- **Client Interval**: Default 30s (demo), production should use 5 minutes
- **Batch Size**: Increase for high throughput (default 100)
- **Kafka Partitions**: Scale to number of unique zipcodes / 100
- **DB Writer Instances**: Can run multiple for higher throughput

## Production Checklist

- [ ] Configure SMTP credentials
- [ ] Set strong DB password
- [ ] Enable Kafka TLS
- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Configure log aggregation
- [ ] Set appropriate connection limits
- [ ] Enable Redis persistence
- [ ] Set up backup strategy for PostgreSQL

---

**Need help?** Check README.md for detailed documentation.

**Ready to scale?** All services can run multiple instances (except aggregator).

