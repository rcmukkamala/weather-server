# Weather Server - Quick Start Guide

**Standalone Deployment on Ubuntu/Linux** - No Kubernetes, No GCP Required!

Get the Weather Server running in 5 minutes as **native Linux processes** on a single machine.

## What This Guide Covers

This is a **standalone deployment** where:
- ✅ Infrastructure runs in Docker containers (PostgreSQL, Redis, Kafka)
- ✅ Application services run as **native Linux processes** (not in containers)
- ✅ Perfect for **development**, **testing**, and **single-machine production**
- ❌ **No Kubernetes required**
- ❌ **No GCP/Cloud required**
- ❌ **No container orchestration**

**For Kubernetes deployment**, see: `DEPLOYMENT_CHECKLIST_3NODE.md`  
**For GCP deployment**, see: `deploy/gcp/README.md`

---

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
- Kafka in KRaft mode (port 9092, 9093)
- Kafka topics auto-initialization (runs once)
- Kafka UI (port 8090)

Wait ~10 seconds for services to be healthy. The Kafka topics (`weather.metrics.raw` and `weather.alarms`) will be automatically created during startup.

## Step 2: Build Services (15 seconds)

```bash
make build
```

Creates 4 binaries in `bin/`:
- `server` - TCP server
- `aggregator` - Hourly/daily aggregation
- `alarming` - Real-time alarm evaluation
- `notification` - Email notifications

## Step 3: Run Services as Native Processes (4 terminals)

**Each service runs as a standalone Linux process** - just like any traditional application!

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

## Deployment Architecture

### This Standalone Setup:
```
┌─────────────────────────────────────────────┐
│         Single Ubuntu Machine               │
├─────────────────────────────────────────────┤
│  Docker Containers (Infrastructure):        │
│  • PostgreSQL (port 5432)                   │
│  • Redis (port 6379)                        │
│  • Kafka in KRaft mode (ports 9092, 9093)  │
│  • Kafka UI (port 8090)                     │
├─────────────────────────────────────────────┤
│  Native Linux Processes (Applications):     │
│  • bin/server - TCP Server (port 8080)     │
│  • bin/aggregator - Aggregation Service     │
│  • bin/alarming - Alarming Service          │
│  • bin/notification - Notification Service  │
└─────────────────────────────────────────────┘
```

### Data Flow:
```
Weather Client (examples/client/main.go)
    ↓ TCP (port 8080)
TCP Server (bin/server) ← Native Linux Process
    ↓ Kafka
[weather.metrics.raw topic]
    ↓                      ↓
DB Writer              Alarming Service ← Native Linux Process
    ↓                      ↓ Redis
PostgreSQL          [weather.alarms topic]
    ↓                      ↓
Aggregation ← Native   Notification Service ← Native
Linux Process          Linux Process
                           ↓ SMTP
```

## Performance Tips

- **Client Interval**: Default 30s (demo), production should use 5 minutes
- **Batch Size**: Increase for high throughput (default 100)
- **Kafka Partitions**: Scale to number of unique zipcodes / 100
- **DB Writer Instances**: Can run multiple for higher throughput

## Stopping Services

To gracefully stop all services:

### Stop Application Processes
Press `Ctrl+C` in each terminal running a service:
1. Terminal 1: Stop TCP Server
2. Terminal 2: Stop Aggregator
3. Terminal 3: Stop Alarming Service
4. Terminal 4: Stop Notification Service

### Stop Infrastructure
```bash
# Stop and remove containers
make docker-down
# or
docker-compose down

# To also remove volumes (deletes data!)
docker-compose down -v
```

## Production Checklist (Standalone)

- [ ] Configure SMTP credentials in `.env`
- [ ] Set strong DB password in `docker-compose.yml` and `.env`
- [ ] Enable Kafka TLS (for production)
- [ ] Set up process management (systemd, supervisor, or pm2)
- [ ] Configure log rotation
- [ ] Set appropriate connection limits
- [ ] Enable Redis persistence and AOF
- [ ] Set up backup strategy for PostgreSQL
- [ ] Consider running services behind nginx/reverse proxy
- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Configure firewall rules (ufw/iptables)

### Optional: Run Services as Systemd Units

For production standalone deployment, create systemd service files:

```bash
# Example: /etc/systemd/system/weather-server.service
[Unit]
Description=Weather Server TCP Service
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=yourusername
WorkingDirectory=/path/to/Weather-Server
ExecStart=/path/to/Weather-Server/bin/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Repeat for aggregator, alarming, and notification services.

---

## Troubleshooting

### Kafka Topics Missing

If you see error: `Unknown Topic Or Partition`, the topics weren't created automatically.

**Manual Fix:**
```bash
make kafka-init
```

**Verify Topics:**
```bash
make kafka-topics
# Should show:
# weather.alarms
# weather.metrics.raw
```

**Reset Everything (if needed):**
```bash
make docker-down
docker volume prune -f
make docker-up
```

### Check Service Health

```bash
# Check all containers
docker ps

# Check Kafka logs
docker logs weather-kafka

# Check if topics exist
docker exec weather-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
```

---

## When to Use This Standalone Deployment

### ✅ Use Standalone When:
- Development and testing
- Single-machine deployment
- Low to medium traffic (< 1000 connections)
- Simple infrastructure requirements
- Cost-sensitive deployments
- Learning the system

### ❌ Use Kubernetes Instead When:
- High availability required (multi-node)
- Auto-scaling needed (based on load)
- High traffic (> 10,000 connections)
- Multi-region deployment
- Need container orchestration
- See: `DEPLOYMENT_CHECKLIST_3NODE.md`

---

**Need help?** Check `README.md` for detailed documentation.

**Ready to scale?** All services can run multiple instances (except aggregator).

**Want production deployment?** See `DEPLOYMENT.md` for Kubernetes and GCP options.

