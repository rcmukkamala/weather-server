# 🚀 Quick Start: 3-Node Kubernetes Deployment

**Fast track to production-ready deployment on 3 worker nodes.**

---

## ⚡ Why 3 Nodes?

✅ **Production-Ready** - Full high availability  
✅ **No Config Changes** - Deploy k8s/kafka.yaml as-is  
✅ **Optimal Kafka** - 3 brokers with RF=3  
✅ **Even Distribution** - Perfect load balancing  
✅ **Fault Tolerant** - Survives 1 node failure  

**Cluster Setup**: 1 Master (+ Worker) + 2 Worker nodes  
**Total Resources**: 12 vCPU, 24GB RAM, 300GB Storage

### Node Distribution

```
Node 1 (Master + Worker):
├── K8s Control Plane + System Pods: ~1.5 CPU, ~3 GB RAM
├── PostgreSQL: 500m CPU, 1 GB RAM
├── Weather Server: 1 replica (250m CPU, 256Mi RAM)
├── Weather Alarming: 1 replica (200m CPU, 256Mi RAM)
└── Kafka Broker (kafka-0): 1 CPU, 2 GB RAM

Node 2 (Worker):
├── Redis: 100m CPU, 256Mi RAM
├── Weather Server: 1 replica (250m CPU, 256Mi RAM)
├── Weather Alarming: 1 replica (200m CPU, 256Mi RAM)
├── Aggregator: 1 replica (100m CPU, 128Mi RAM)
├── Notification: 1 replica (100m CPU, 128Mi RAM)
└── Kafka Broker (kafka-1): 1 CPU, 2 GB RAM

Node 3 (Worker):
├── Weather Server: 1 replica (250m CPU, 256Mi RAM)
├── Notification: 1 replica (100m CPU, 128Mi RAM)
└── Kafka Broker (kafka-2): 1 CPU, 2 GB RAM
```

---

## 🎯 3 Simple Steps

### Step 1: Build & Push (5 minutes)

```bash
export REGISTRY="your-registry.io/weather"
export VERSION="v1.0.0"

# Build all 4 services
docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:${VERSION} .
docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:${VERSION} .
docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:${VERSION} .
docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:${VERSION} .

# Push
docker push ${REGISTRY}/weather-server:${VERSION}
docker push ${REGISTRY}/weather-aggregator:${VERSION}
docker push ${REGISTRY}/weather-alarming:${VERSION}
docker push ${REGISTRY}/weather-notification:${VERSION}
```

### Step 2: Configure (2 minutes)

```bash
# 1. Edit secrets (REQUIRED!)
vi k8s/secrets.yaml
# Change DB_PASSWORD from "CHANGE_ME_STRONG_PASSWORD" to your password

# 2. Update image registry in all deployment files
find k8s/ -name "*-deployment.yaml" -exec \
  sed -i "s|gcr.io/YOUR_PROJECT_ID|${REGISTRY}|g" {} +

# 3. Verify
grep "image:" k8s/server-deployment.yaml
# Should show YOUR registry
```

### Step 3: Deploy (25 minutes)

```bash
# Infrastructure
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/kafka.yaml  # ✓ NO EDITS NEEDED!

# Wait for infrastructure (5-10 min)
kubectl wait --for=condition=ready pod -l app=postgres -n weather-system --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n weather-system --timeout=300s
kubectl wait --for=condition=ready pod -l app=kafka -n weather-system --timeout=600s

# Application services
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/aggregator-deployment.yaml
kubectl apply -f k8s/alarming-deployment.yaml
kubectl apply -f k8s/notification-deployment.yaml

# Wait for apps (2-3 min)
kubectl wait --for=condition=available deployment --all -n weather-system --timeout=300s
```

---

## ✅ Verify Deployment

```bash
# All pods running (should be 15-17 pods)
kubectl get pods -n weather-system

# Expected:
# postgres-0                      1/1   Running
# redis-0                         1/1   Running
# kafka-0, kafka-1, kafka-2      3/3   Running  ✓ (KRaft mode)
# weather-server-xxx (×3)        3/3   Running
# weather-aggregator-xxx          1/1   Running
# weather-alarming-xxx (×2)       2/2   Running
# weather-notification-xxx (×2)   2/2   Running

# Check Kafka has 3 brokers
kubectl get pods -n weather-system -l app=kafka
# Should show kafka-0, kafka-1, kafka-2

# Get external endpoint
kubectl get svc weather-server-service -n weather-system
# Note the EXTERNAL-IP or NodePort

# Test connection
nc -zv <EXTERNAL-IP> 8080
# Connection succeeded!
```

---

## 🧪 Test with Sample Client

```bash
# Terminal 1: Run client
go run examples/client/main.go

# Expected output:
# ✓ Connected to server
# → Sent identify message
# ← Received ack: identified
# → Sent metrics: temp=25.3°C...
```

```bash
# Terminal 2: Verify data in database
kubectl exec -n weather-system statefulset/postgres -- \
  psql -U weather_user -d weather_db -c \
  "SELECT COUNT(*) FROM raw_metrics;"
# Should show increasing count
```

---

## 📊 What You Get

| Component | Replicas | Resources | High Availability |
|-----------|----------|-----------|-------------------|
| **Kafka Brokers** | 3 | 3 CPUs, 6GB RAM | ✅ Yes (RF=3) |
| **TCP Server** | 3 (scales to 10) | 750m CPU, 768Mi RAM | ✅ Yes |
| **Alarming Service** | 2 (scales to 5) | 400m CPU, 512Mi RAM | ✅ Yes |
| **Notification** | 2 | 200m CPU, 256Mi RAM | ✅ Yes |
| **PostgreSQL** | 1 | 500m CPU, 1Gi RAM | ⚠️ StatefulSet |
| **Redis** | 1 | 100m CPU, 256Mi RAM | ⚠️ StatefulSet |

**Total Used**: ~6-8 CPUs, ~10-13GB RAM  
**Total Available**: 12 CPUs, 24GB RAM  
**Utilization**: 50-65% (comfortable headroom)

---

## 🎛️ Scale Up (Optional)

```bash
# Scale TCP server to 6 replicas (2 per node)
kubectl scale deployment weather-server --replicas=6 -n weather-system

# Scale alarming to 3 replicas (1 per node)
kubectl scale deployment weather-alarming --replicas=3 -n weather-system

# Verify distribution
kubectl get pods -n weather-system -o wide
# Pods should be evenly distributed across 3 nodes
```

---

## 🔧 Key Differences from 2-Node Setup

| Feature | 2-Node | 3-Node |
|---------|--------|--------|
| **Kafka Edits** | ⚠️ REQUIRED | ✅ NOT REQUIRED |
| **k8s/kafka.yaml** | Must change replicas to 2 | ✓ Use as-is |
| **Replication Factor** | Must change to 2 | ✓ Stays at 3 |
| **Brokers** | 2 (kafka-0, kafka-1) | 3 (kafka-0, kafka-1, kafka-2) |
| **High Availability** | Limited | Full |
| **Cost** | $260/mo | $390/mo |
| **Best For** | Dev/Staging | **Production** ✅ |

---

## 🆘 Troubleshooting

### Pods Not Starting?
```bash
kubectl describe pod <pod-name> -n weather-system
kubectl logs <pod-name> -n weather-system
```

### Kafka Taking Long to Start?
```bash
# This is normal! Kafka cluster formation takes 5-10 minutes
kubectl logs kafka-0 -n weather-system --tail=50
# Look for: "started (kafka.server.KafkaServer)"
```

### Can't Connect Externally?
```bash
# If LoadBalancer is pending, use NodePort:
kubectl patch svc weather-server-service -n weather-system \
  -p '{"spec":{"type":"NodePort"}}'

NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
NODE_PORT=$(kubectl get svc weather-server-service -n weather-system \
  -o jsonpath='{.spec.ports[0].nodePort}')
echo "Connect to: ${NODE_IP}:${NODE_PORT}"
```

---

## 📋 Full Checklist

For a comprehensive step-by-step guide with 155+ verification items:

👉 **[DEPLOYMENT_CHECKLIST_3NODE.md](DEPLOYMENT_CHECKLIST_3NODE.md)**

---

## 💰 Cost Breakdown (AWS Example)

| Item | Spec | Monthly Cost |
|------|------|--------------|
| 3 × t3.xlarge nodes | 4 vCPU, 16GB RAM each | $364 |
| 3 × 100GB EBS volumes | gp3 SSD | $24 |
| Load Balancer | Network LB | $16 |
| **Total** | | **~$404/month** |

Compare to 2-node: **$130/month more for 50% more capacity**

---

## ✨ Next Steps

1. ✅ Deploy using steps above
2. ✅ Verify all 15-17 pods running
3. ✅ Test with sample client
4. ✅ Check metrics in database
5. 🔜 Configure alarm thresholds (see [scripts/setup_alarms.sql](scripts/setup_alarms.sql))
6. 🔜 Setup monitoring (Prometheus/Grafana)
7. 🔜 Configure email notifications (update secrets)

---

## 📚 Related Documents

- **Detailed 3-Node Checklist**: [DEPLOYMENT_CHECKLIST_3NODE.md](DEPLOYMENT_CHECKLIST_3NODE.md)
- **2-Node vs 3-Node Comparison**: [DEPLOYMENT_COMPARISON.md](DEPLOYMENT_COMPARISON.md)
- **2-Node Deployment**: [DEPLOYMENT_CHECKLIST_2NODE.md](DEPLOYMENT_CHECKLIST_2NODE.md)
- **Main Deployment Guide**: [DEPLOYMENT.md](DEPLOYMENT.md)
- **Architecture Overview**: [README.md](README.md)

---

**🎉 You're deploying to production! The 3-node setup gives you enterprise-grade reliability with minimal configuration.**

**Deployment Time**: 30-60 minutes  
**Difficulty**: Medium  
**Production Ready**: ✅ Yes

