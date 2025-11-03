# 🏗️ Weather Server - 3-Node Kubernetes Architecture

Production-ready cluster architecture with optimal resource distribution.

---

## 🎯 Cluster Overview

**Setup**: 1 Master + 2 Worker Nodes (Master also acts as Worker)  
**Total Resources**: 12 vCPU, 24 GB RAM, 300 GB Storage  
**High Availability**: ✅ Full (Kafka RF=3, Service Redundancy)  
**Cost**: ~$390/month (AWS t3.xlarge)

---

## 📊 Node Distribution

### Node 1 (Master + Worker)
```
Hostname: k8s-master-01
Role: Control Plane + Worker
Resources: 4 vCPU, 8 GB RAM, 100 GB SSD

Workloads:
├── Kubernetes Control Plane
│   ├── kube-apiserver
│   ├── kube-scheduler
│   ├── kube-controller-manager
│   └── etcd
│   Resource Usage: ~1 CPU, ~2 GB RAM
│
├── Kubernetes System Pods
│   ├── kube-proxy
│   ├── coredns
│   ├── metrics-server
│   └── calico/flannel (CNI)
│   Resource Usage: ~0.5 CPU, ~1 GB RAM
│
├── PostgreSQL StatefulSet (postgres-0)
│   ├── Request: 500m CPU, 1 GB RAM
│   ├── Limit: 1 CPU, 2 GB RAM
│   ├── PVC: 20 GB
│   └── Port: 5432
│
├── Weather Server (weather-server-abc123)
│   ├── Request: 250m CPU, 256 Mi RAM
│   ├── Limit: 500m CPU, 512 Mi RAM
│   └── Replica 1 of 3
│
├── Alarming Service (weather-alarming-def456)
│   ├── Request: 200m CPU, 256 Mi RAM
│   ├── Limit: 400m CPU, 512 Mi RAM
│   └── Replica 1 of 2
│
└── Kafka Broker (kafka-0)
    ├── Request: 1 CPU, 2 GB RAM
    ├── Limit: 2 CPU, 4 GB RAM
    ├── PVC: 50 GB
    └── Broker ID: 0

Total Node Usage: ~3.5 CPU, ~6.5 GB RAM (Available: 0.5 CPU, 1.5 GB RAM)
```

---

### Node 2 (Worker)
```
Hostname: k8s-worker-01
Role: Worker Node
Resources: 4 vCPU, 8 GB RAM, 100 GB SSD

Workloads:
├── Redis StatefulSet (redis-0)
│   ├── Request: 100m CPU, 256 Mi RAM
│   ├── Limit: 250m CPU, 512 Mi RAM
│   ├── PVC: 10 GB
│   └── Port: 6379
│
├── Weather Server (weather-server-ghi789)
│   ├── Request: 250m CPU, 256 Mi RAM
│   ├── Limit: 500m CPU, 512 Mi RAM
│   └── Replica 2 of 3
│
├── Alarming Service (weather-alarming-jkl012)
│   ├── Request: 200m CPU, 256 Mi RAM
│   ├── Limit: 400m CPU, 512 Mi RAM
│   └── Replica 2 of 2
│
├── Aggregator Service (weather-aggregator-mno345)
│   ├── Request: 100m CPU, 128 Mi RAM
│   ├── Limit: 200m CPU, 256 Mi RAM
│   └── Single instance
│
├── Notification Service (weather-notification-pqr678)
│   ├── Request: 100m CPU, 128 Mi RAM
│   ├── Limit: 200m CPU, 256 Mi RAM
│   └── Replica 1 of 2
│
└── Kafka Broker (kafka-1)
    ├── Request: 1 CPU, 2 GB RAM
    ├── Limit: 2 CPU, 4 GB RAM
    ├── PVC: 50 GB
    └── Broker ID: 1

Total Node Usage: ~2 CPU, ~3 GB RAM (Available: 2 CPU, 5 GB RAM)
```

---

### Node 3 (Worker)
```
Hostname: k8s-worker-02
Role: Worker Node
Resources: 4 vCPU, 8 GB RAM, 100 GB SSD

Workloads:
│   ├── PVC: 10 GB
│   └── Port: 2181
│
├── Weather Server (weather-server-stu901)
│   ├── Request: 250m CPU, 256 Mi RAM
│   ├── Limit: 500m CPU, 512 Mi RAM
│   └── Replica 3 of 3
│
├── Notification Service (weather-notification-vwx234)
│   ├── Request: 100m CPU, 128 Mi RAM
│   ├── Limit: 200m CPU, 256 Mi RAM
│   └── Replica 2 of 2
│
└── Kafka Broker (kafka-2)
    ├── Request: 1 CPU, 2 GB RAM
    ├── Limit: 2 CPU, 4 GB RAM
    ├── PVC: 50 GB
    └── Broker ID: 2

Total Node Usage: ~2 CPU, ~3 GB RAM (Available: 2 CPU, 5 GB RAM)
```

---

## 📈 Cluster Resource Summary

### Total Resource Allocation

| Resource Type | Total Requested | Total Available | Utilization |
|---------------|----------------|-----------------|-------------|
| **CPU** | 7.45 cores | 12 cores | 62% |
| **Memory** | 12.5 GB | 24 GB | 52% |
| **Storage (PVC)** | 140 GB | 300 GB | 47% |
| **Pods** | 15 pods | 330 pods | 5% |

### Per-Node Breakdown

| Node | CPU Used | CPU Available | Memory Used | Memory Available | Headroom |
|------|----------|---------------|-------------|------------------|----------|
| Node 1 (Master + Worker) | 3.5 / 4 | 0.5 vCPU | 6.5 / 8 GB | 1.5 GB | 12-20% |
| Node 2 (Worker) | 2 / 4 | 2 vCPU | 3 / 8 GB | 5 GB | 50% |
| Node 3 (Worker) | 2 / 4 | 2 vCPU | 3 / 8 GB | 5 GB | 50% |

---

## 🔄 Service Connectivity

### Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    External Clients                          │
│                 (Weather Stations / IoT)                     │
└───────────────────────┬─────────────────────────────────────┘
                        │ TCP:8080
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              LoadBalancer / NodePort                         │
│         weather-server-service (External)                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌────────┐     ┌────────┐     ┌────────┐
   │Server-1│     │Server-2│     │Server-3│  (3 replicas)
   │ Node 1 │     │ Node 2 │     │ Node 3 │
   └────┬───┘     └────┬───┘     └────┬───┘
        │              │              │
        └──────────────┼──────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  Kafka Cluster (3 brokers)   │
        │  kafka-0, kafka-1, kafka-2   │
        │  Topic: weather.metrics.raw  │
        │  Partitions: 10, RF: 3       │
        └───────┬──────────────┬───────┘
                │              │
        ┌───────▼───┐   ┌──────▼────────┐
        │           │   │               │
   ┌────▼─────┐  ┌─▼──────┐  ┌─────────▼───┐
   │Batch     │  │Alarming│  │Notification │
   │Writer    │  │Service │  │Service      │
   │(Server)  │  │(2 reps)│  │(2 replicas) │
   └────┬─────┘  └───┬────┘  └──────────── ┘
        │            │
        │            ▼
        │       ┌────────┐
        │       │ Redis  │  (Alarm state cache)
        │       │ Node 2 │
        │       └────────┘
        │
        ▼
   ┌────────────┐
   │ PostgreSQL │  (Time-series data)
   │   Node 1   │
   └──────┬─────┘
          │
          ▼
   ┌──────────────┐
   │ Aggregator   │  (Hourly/Daily jobs)
   │   Node 2     │
   └──────────────┘
```

---

## 🌐 Network Architecture

### Service Mesh

| Service | Type | Internal DNS | Port | Load Balancing |
|---------|------|--------------|------|----------------|
| **weather-server-service** | LoadBalancer | weather-server-service.weather-system.svc.cluster.local | 8080 | External LB |
| **postgres-service** | ClusterIP | postgres-service.weather-system.svc.cluster.local | 5432 | Round-robin |
| **redis-service** | ClusterIP | redis-service.weather-system.svc.cluster.local | 6379 | Single endpoint |
| **kafka-service** | ClusterIP | kafka-service.weather-system.svc.cluster.local | 9092 | Kafka protocol (KRaft) |

### Internal Communication

- **TCP Server → Kafka**: Produces metrics to `weather.metrics.raw` topic
- **TCP Server → PostgreSQL**: Batch writes via Kafka consumer
- **Alarming Service → Redis**: Read/Write alarm states
- **Alarming Service → PostgreSQL**: Read thresholds, write alarm logs
- **Alarming Service → Kafka**: Produces to `weather.alarms` topic
- **Notification Service → Kafka**: Consumes from `weather.alarms` topic
- **Aggregator → PostgreSQL**: Read raw metrics, write aggregated data

---

## 🔐 Storage Architecture

### Persistent Volumes

| PVC Name | Size | Mount Path | Used By | Node Affinity |
|----------|------|------------|---------|---------------|
| **postgres-data-postgres-0** | 20 GB | /var/lib/postgresql/data | postgres-0 | Node 1 |
| **redis-data-redis-0** | 10 GB | /data | redis-0 | Node 2 |
| **kafka-data-kafka-0** | 50 GB | /var/lib/kafka/data | kafka-0 | Node 1 |
| **kafka-data-kafka-1** | 50 GB | /var/lib/kafka/data | kafka-1 | Node 2 |
| **kafka-data-kafka-2** | 50 GB | /var/lib/kafka/data | kafka-2 | Node 3 |

**Total PVC Usage**: 130 GB  
**Storage Class**: Default (or specify gp3, pd-ssd, etc.)

---

## 🚀 Auto-Scaling Configuration

### Horizontal Pod Autoscaler (HPA)

#### Weather Server HPA
```yaml
Min Replicas: 3
Max Replicas: 10
Target CPU: 70%
Target Memory: 70%
Scale Up: +3 pods if CPU > 70% for 1 min
Scale Down: -1 pod if CPU < 50% for 5 min
```

#### Alarming Service HPA
```yaml
Min Replicas: 2
Max Replicas: 5
Target CPU: 70%
Target Memory: 70%
Scale Up: +1 pod if CPU > 70% for 1 min
Scale Down: -1 pod if CPU < 50% for 5 min
```

### Cluster Autoscaler (Optional)
- Can add nodes 4-10 based on pending pods
- Scale down after 10 min of low utilization
- Respects PodDisruptionBudgets

---

## 🛡️ High Availability Features

### Service Level HA

| Component | Replicas | Failure Tolerance | Recovery Time |
|-----------|----------|-------------------|---------------|
| **TCP Server** | 3-10 | 2 pods can fail | < 30 sec (HPA) |
| **Alarming Service** | 2-5 | 1 pod can fail | < 30 sec (HPA) |
| **Notification Service** | 2 | 1 pod can fail | < 60 sec |
| **Kafka Cluster** | 3 | 1 broker can fail | Immediate |
| **PostgreSQL** | 1 | Node must survive | Manual failover |
| **Redis** | 1 | Node must survive | Data in memory |

### Kafka High Availability

- **Replication Factor**: 3 (every message replicated 3x)
- **Min In-Sync Replicas**: 2 (requires 2 brokers to acknowledge)
- **Leader Election**: Automatic via KRaft (built-in consensus)
- **Partition Distribution**: Even across 3 brokers
- **Failure Scenario**: 
  - 1 broker fails → Service continues, elections occur
  - 2 brokers fail → Service degraded, read-only possible
  - 3 brokers fail → Service down

### Node Failure Scenarios

#### Node 1 (Master + Worker) Fails
- **Control Plane**: Cluster loses management (CRITICAL)
- **PostgreSQL**: Database unavailable until node recovers
- **Kafka-0**: Kafka re-elects leaders, continues with 2 brokers
- **Services**: Pods reschedule to Node 2 and Node 3
- **Recovery**: Manual intervention required for control plane

#### Node 2 (Worker) Fails
- **Control Plane**: Unaffected ✅
- **Redis**: Cache unavailable until node recovers
- **Kafka-1**: Kafka continues with 2 brokers
- **Services**: Pods reschedule to Node 1 and Node 3
- **Recovery**: Automatic (pods reschedule within 60 sec)

#### Node 3 (Worker) Fails
- **Control Plane**: Unaffected ✅
- **Kafka-2**: Kafka continues with 2 brokers (quorum maintained via KRaft)
- **Services**: Pods reschedule to Node 1 and Node 2
- **Recovery**: Automatic (pods reschedule within 60 sec)

---

## 📊 Monitoring & Observability

### Health Checks

| Service | Liveness Probe | Readiness Probe | Startup Time |
|---------|----------------|-----------------|--------------|
| TCP Server | TCP :8080 | TCP :8080 | 10 sec |
| Aggregator | - | - | 5 sec |
| Alarming | - | - | 10 sec |
| Notification | - | - | 5 sec |
| PostgreSQL | `pg_isready` | `SELECT 1` | 30 sec |
| Redis | `redis-cli ping` | `PING` | 5 sec |
| Kafka | Port 9092 | Bootstrap check | 60 sec |

### Key Metrics to Monitor

**Application Metrics:**
- TCP connections active
- Messages per second (Kafka)
- Alarm evaluations per second
- Email notifications sent
- Database query latency

**Infrastructure Metrics:**
- Node CPU/Memory utilization
- Pod restart count
- PVC disk usage
- Network throughput
- Kafka consumer lag

---

## 💰 Cost Breakdown (AWS Example)

### Compute (EC2)
```
Node 1: t3.xlarge (4 vCPU, 16 GB) × 730 hours = $121.47/month
Node 2: t3.xlarge (4 vCPU, 16 GB) × 730 hours = $121.47/month
Node 3: t3.xlarge (4 vCPU, 16 GB) × 730 hours = $121.47/month
Total Compute: $364.41/month
```

### Storage (EBS gp3)
```
Node 1: 100 GB × $0.08/GB = $8.00/month
Node 2: 100 GB × $0.08/GB = $8.00/month
Node 3: 100 GB × $0.08/GB = $8.00/month
Total Storage: $24.00/month
```

### Network
```
Load Balancer (NLB): $16.20/month
Data Transfer: ~$10-50/month (depending on traffic)
Total Network: ~$30/month
```

**Monthly Total: ~$420/month**

---

## 🎯 Production Optimizations

### Recommended Enhancements

1. **Multi-Zone Deployment**
   - Spread nodes across 3 availability zones
   - Add node affinity rules
   - Cost: +0% (same node count)

2. **PostgreSQL Replication**
   - Deploy PostgreSQL with streaming replication
   - 1 primary + 1 standby
   - Cost: +$120/month (1 extra node)

3. **Redis Sentinel**
   - Deploy Redis with Sentinel HA
   - 1 master + 2 replicas + 3 sentinels
   - Cost: +$0 (fits on existing nodes)

4. **Monitoring Stack**
   - Prometheus + Grafana + Alertmanager
   - Cost: +$60/month (t3.medium)

5. **Log Aggregation**
   - ELK/EFK stack or managed logging
   - Cost: +$100/month

**Total Production-Grade Cost**: ~$700-800/month

---

## 📚 Related Documentation

- **Deployment Checklist**: [DEPLOYMENT_CHECKLIST_3NODE.md](DEPLOYMENT_CHECKLIST_3NODE.md)
- **Quick Start Guide**: [3NODE_QUICK_START.md](3NODE_QUICK_START.md)
- **Deployment Comparison**: [DEPLOYMENT_COMPARISON.md](DEPLOYMENT_COMPARISON.md)
- **Main README**: [README.md](README.md)

---

**This architecture provides production-grade reliability, performance, and scalability for your Weather Server.**

**Cluster Characteristics:**
- ✅ Production-Ready
- ✅ High Availability (with limitations on StatefulSets)
- ✅ Auto-Scaling enabled
- ✅ Fault-Tolerant (1 node failure)
- ✅ Resource-Efficient (60% utilization)
- ✅ Cost-Optimized (~$390-420/month)

