# Weather Server Deployment Guide

Quick reference for deploying the Weather Server to Kubernetes and GCP.

## 🚀 Quick Deployment Options

### Option 1: Local Development (Docker Compose)
```bash
make docker-up
make build
make run-server    # Terminal 1
make run-aggregator    # Terminal 2
make run-alarming      # Terminal 3
make run-notification  # Terminal 4
```
**Time**: 5 minutes  
**Cost**: Free  
**Best for**: Development, testing

---

### Option 2: Kubernetes (Any Provider)

**Choose Your Cluster Size:**

#### 🎯 3-Node Cluster (RECOMMENDED for Production)
✅ Full high availability  
✅ No Kafka configuration changes needed  
✅ Perfect for production workloads  
**Checklist**: [DEPLOYMENT_CHECKLIST_3NODE.md](DEPLOYMENT_CHECKLIST_3NODE.md)

```bash
# Requirements: 3 nodes × 4 vCPU × 8GB RAM = 12 vCPU, 24GB RAM
# Build images
export REGISTRY="your-registry.io/weather"
docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:v1.0.0 .
docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:v1.0.0 .
docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:v1.0.0 .
docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:v1.0.0 .
docker push ${REGISTRY}/weather-server:v1.0.0
docker push ${REGISTRY}/weather-aggregator:v1.0.0
docker push ${REGISTRY}/weather-alarming:v1.0.0
docker push ${REGISTRY}/weather-notification:v1.0.0

# Deploy (NO edits needed to k8s/kafka.yaml!)
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secrets.yaml  # Edit first!
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/kafka.yaml  # ✓ Use as-is
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/aggregator-deployment.yaml
kubectl apply -f k8s/alarming-deployment.yaml
kubectl apply -f k8s/notification-deployment.yaml
```
**Time**: 30-60 minutes  
**Cost**: ~$390/month (AWS)  
**Best for**: Production environments  

#### 💰 2-Node Cluster (Budget Option)
⚠️ Requires Kafka configuration edits  
⚠️ Limited high availability  
✅ Good for dev/staging  
**Checklist**: [DEPLOYMENT_CHECKLIST_2NODE.md](DEPLOYMENT_CHECKLIST_2NODE.md)

```bash
# Requirements: 2 nodes × 4 vCPU × 8GB RAM = 8 vCPU, 16GB RAM
# CRITICAL: Edit k8s/kafka.yaml BEFORE deployment
# Change: replicas: 3 → 2
# Change: replication factors: 3 → 2
# See DEPLOYMENT_CHECKLIST_2NODE.md for details

# Then deploy as above
```
**Time**: 30-45 minutes  
**Cost**: ~$260/month (AWS)  
**Best for**: Development, staging, budget-constrained environments  

**Need Help Choosing?** See [DEPLOYMENT_COMPARISON.md](DEPLOYMENT_COMPARISON.md)

**Detailed Generic Guide**: [deploy/kubernetes/README.md](deploy/kubernetes/README.md)

---

### Option 3: Google Cloud Platform (Managed Services)
```bash
# Setup
export PROJECT_ID="your-gcp-project"
export REGION="us-central1"
gcloud config set project ${PROJECT_ID}

# Create infrastructure
gcloud container clusters create weather-cluster --region ${REGION}
gcloud sql instances create weather-postgres --tier=db-custom-2-8192 --region=${REGION}
gcloud redis instances create weather-redis --size=5 --region=${REGION}

# Build and deploy
./deploy/gcp/deploy.sh v1.0.0
```
**Time**: 45 minutes  
**Cost**: ~$900-1,500/month  
**Best for**: Production on GCP with managed services  
**Detailed Guide**: [deploy/gcp/README.md](deploy/gcp/README.md)

---

## 📦 What Gets Deployed

### 4 Microservices
1. **TCP Server** (3-10 replicas)
   - Handles client connections
   - Publishes to Kafka
   - Auto-scales based on CPU/memory

2. **Aggregation Service** (1 replica)
   - Hourly aggregation (HH:05:00)
   - Daily aggregation (00:05:00)
   - Scheduled tasks

3. **Alarming Service** (2-5 replicas)
   - Real-time threshold monitoring
   - Redis state management
   - Auto-scales with traffic

4. **Notification Service** (2 replicas)
   - Email alerts
   - SMTP delivery
   - Retry logic

### Infrastructure Dependencies
- **PostgreSQL**: Time-series metrics storage
- **Redis**: Alarm state cache
- **Kafka**: Event streaming (10 partitions)
- **Load Balancer**: TCP ingress

---

## 🔧 Pre-Deployment Checklist

### Required
- [ ] Docker registry access
- [ ] Kubernetes cluster (1.24+)
- [ ] Database (PostgreSQL 15+)
- [ ] Redis instance
- [ ] Kafka cluster
- [ ] Container images built and pushed

### Configuration
- [ ] Update `k8s/secrets.yaml` with actual credentials
- [ ] Update `k8s/configmap.yaml` with service endpoints
- [ ] Update deployment YAMLs with your registry path
- [ ] Configure SMTP settings (or leave empty for log-only)

### Security
- [ ] Strong database password
- [ ] Redis authentication (if enabled)
- [ ] Kafka TLS/SASL (for production)
- [ ] Kubernetes RBAC configured
- [ ] Network policies (recommended)

---

## 📊 Resource Requirements

### Minimum (Development - Local Docker Compose)
- **CPUs**: 4 cores
- **Memory**: 8 GB RAM
- **Storage**: 50 GB

### Kubernetes 2-Node Cluster (Budget/Staging)
- **Nodes**: 2 worker nodes
- **CPUs**: 8 vCPUs (4 per node)
- **Memory**: 16 GB RAM (8 per node)
- **Storage**: 200 GB (100 per node)
- **Network**: 1 Gbps
- **Cost**: ~$260/month (AWS)

### Kubernetes 3-Node Cluster (Production - RECOMMENDED)
- **Nodes**: 3 worker nodes
- **CPUs**: 12 vCPUs (4 per node)
- **Memory**: 24 GB RAM (8 per node)
- **Storage**: 300 GB (100 per node)
- **Network**: 1 Gbps
- **Cost**: ~$390/month (AWS)

### Per-Service Requirements
| Service | CPU (Request/Limit) | Memory (Request/Limit) | Replicas (2-node) | Replicas (3-node) |
|---------|---------------------|------------------------|-------------------|-------------------|
| TCP Server | 250m/500m | 256Mi/512Mi | 2-4 | 3-6 |
| Aggregator | 100m/200m | 128Mi/256Mi | 1 | 1 |
| Alarming | 200m/400m | 256Mi/512Mi | 2 | 2-3 |
| Notification | 100m/200m | 128Mi/256Mi | 2 | 2-3 |
| PostgreSQL | 500m/1000m | 1Gi/2Gi | 1 | 1 |
| Redis | 100m/250m | 256Mi/512Mi | 1 | 1 |
| Kafka (KRaft) | 1000m/2000m | 2Gi/4Gi | 2 | 3 |

---

## 🎯 Deployment Architectures

### Development Architecture (Local)
```
Local Docker Compose
├── PostgreSQL (single instance)
├── Redis (single instance)
├── Kafka (single broker)
└── 4 Services (1 instance each)
```

### Kubernetes 2-Node Architecture (Budget/Staging)
```
2-Node K8s Cluster (8 vCPU, 16GB RAM)
├── Node 1:
│   ├── PostgreSQL (1 replica)
│   ├── Weather Server (1-2 replicas)
│   ├── Kafka (1 broker)
│   └── System Pods
└── Node 2:
    ├── Redis (1 replica)
    ├── Weather Server (1-2 replicas)
    ├── Alarming Service (2 replicas)
    ├── Notification Service (2 replicas)
    ├── Aggregator (1 replica)
    └── Kafka in KRaft mode (2 brokers)

⚠️ Requires: Kafka replicas=2, replication_factor=2
```

### Kubernetes 3-Node Architecture (Production - RECOMMENDED)
```
3-Node K8s Cluster (12 vCPU, 24GB RAM)

Node 1 (Master + Worker):
├── K8s Control Plane: ~1 CPU, ~2 GB RAM
├── K8s System Pods: ~0.5 CPU, ~1 GB RAM
├── PostgreSQL: 500m CPU, 1 GB RAM
├── Weather Server: 1 replica (250m CPU, 256Mi)
├── Alarming Service: 1 replica (200m CPU, 256Mi)
└── Kafka Broker (kafka-0): 1 CPU, 2 GB RAM
    Total: ~3.5 CPU, ~6.5 GB RAM

Node 2 (Worker):
├── Redis: 100m CPU, 256Mi RAM
├── Weather Server: 1 replica (250m CPU, 256Mi)
├── Alarming Service: 1 replica (200m CPU, 256Mi)
├── Aggregator: 1 replica (100m CPU, 128Mi)
├── Notification: 1 replica (100m CPU, 128Mi)
└── Kafka Broker (kafka-1): 1 CPU, 2 GB RAM
    Total: ~2 CPU, ~3 GB RAM

Node 3 (Worker):
├── Weather Server: 1 replica (250m CPU, 256Mi)
├── Notification: 1 replica (100m CPU, 128Mi)
└── Kafka Broker (kafka-2 in KRaft mode): 1 CPU, 2 GB RAM
    Total: ~1.5 CPU, ~2.5 GB RAM

✅ No ZooKeeper required (Kafka 4.0+ uses KRaft mode)
✅ Full high availability with 3 Kafka brokers (RF=3)
✅ Optimal pod distribution across all nodes
✅ Control plane co-located with worker on Node 1
```

### GCP Production Architecture (Managed Services)
```
GCP
├── GKE Cluster (3+ nodes, multi-zone)
│   └── Deployments: 4 Services
├── Cloud SQL (PostgreSQL HA)
├── Memorystore Redis (Standard tier)
├── Confluent Cloud Kafka / Managed Kafka
└── Cloud Load Balancer
```

---

## 🔐 Security Best Practices

### Secrets Management
**Development:**
```bash
kubectl create secret generic weather-secrets \
  --from-literal=DB_PASSWORD='strong_password' \
  --namespace=weather-system
```

**Production (GCP):**
```bash
# Use Secret Manager
gcloud secrets create db-password --data-file=-
# Integrate with Workload Identity
```

### Network Security
```yaml
# Example NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: weather-network-policy
spec:
  podSelector:
    matchLabels:
      app: weather-server
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: weather-system
```

### TLS/SSL
- Enable TLS for Kafka (production)
- Use Cloud SQL Proxy for secure DB connections
- Configure SMTP with TLS (port 587)

---

## 📈 Monitoring Setup

### Prometheus Metrics
```bash
# Install Prometheus Operator
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml

# Create ServiceMonitor
kubectl apply -f deploy/monitoring/servicemonitor.yaml
```

### Cloud Monitoring (GCP)
Metrics automatically sent to Cloud Monitoring:
- CPU/Memory usage
- Request latency
- Error rates
- Custom metrics

### Logging
```bash
# Kubernetes
kubectl logs -f deployment/weather-server -n weather-system

# GCP
gcloud logging read "resource.type=k8s_container" --limit=50
```

---

## 🔄 CI/CD Pipelines

### GitHub Actions
```yaml
# .github/workflows/deploy.yml
name: Deploy to Kubernetes
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build and Push
        run: |
          docker build -f Dockerfile.server -t ${{ secrets.REGISTRY }}/weather-server:${{ github.sha }} .
          docker push ${{ secrets.REGISTRY }}/weather-server:${{ github.sha }}
      - name: Deploy
        run: kubectl set image deployment/weather-server server=${{ secrets.REGISTRY }}/weather-server:${{ github.sha }}
```

### Cloud Build (GCP)
```bash
# Trigger automated build and deploy
gcloud builds submit --config=deploy/gcp/cloudbuild.yaml
```

---

## 🆘 Troubleshooting

### Pods Not Starting
```bash
kubectl describe pod <pod-name> -n weather-system
kubectl logs <pod-name> -n weather-system --previous
```

### Database Connection Issues
```bash
# Test connection from pod
kubectl exec -it deployment/weather-server -n weather-system -- /bin/sh
nc -zv postgres-service 5432
```

### Kafka Connection Issues
```bash
# Check Kafka logs
kubectl logs statefulset/kafka -n weather-system

# Test from pod
kubectl exec -it deployment/weather-server -n weather-system -- /bin/sh
nc -zv kafka-service 9092
```

---

## 📚 Additional Resources

### Deployment Guides
- **3-Node Deployment Checklist**: [DEPLOYMENT_CHECKLIST_3NODE.md](DEPLOYMENT_CHECKLIST_3NODE.md) ⭐ **Production**
- **2-Node Deployment Checklist**: [DEPLOYMENT_CHECKLIST_2NODE.md](DEPLOYMENT_CHECKLIST_2NODE.md) - Budget/Staging
- **Deployment Comparison**: [DEPLOYMENT_COMPARISON.md](DEPLOYMENT_COMPARISON.md) - Choose Your Setup
- **Kubernetes Generic Guide**: [deploy/kubernetes/README.md](deploy/kubernetes/README.md)
- **GCP Guide**: [deploy/gcp/README.md](deploy/gcp/README.md)
- **Deployment Summary**: [DEPLOYMENT_SUMMARY.md](DEPLOYMENT_SUMMARY.md)

### Project Documentation
- **Main README**: [README.md](README.md)
- **Quick Start**: [QUICKSTART.md](QUICKSTART.md)

---

## 🎓 Support Matrix

| Platform | Support Level | Documentation |
|----------|---------------|---------------|
| Docker Compose | ✅ Full | [README.md](README.md) |
| Kubernetes (Generic) | ✅ Full | [deploy/kubernetes/README.md](deploy/kubernetes/README.md) |
| Google Cloud (GKE) | ✅ Full | [deploy/gcp/README.md](deploy/gcp/README.md) |
| AWS (EKS) | 🟡 Community | Adapt K8s guide |
| Azure (AKS) | 🟡 Community | Adapt K8s guide |
| OpenShift | 🟡 Community | Adapt K8s guide |

---

## 💰 Cost Estimates

### GCP (Production)
- **Compute (GKE)**: $150-750/month
- **Database (Cloud SQL)**: $250/month
- **Cache (Memorystore)**: $130/month
- **Messaging (Kafka)**: $300/month
- **Load Balancer**: $20/month
- **Total**: **~$850-1,450/month**

### AWS (Production)
- **Compute (EKS)**: $150-750/month
- **Database (RDS)**: $200/month
- **Cache (ElastiCache)**: $100/month
- **Messaging (MSK)**: $250/month
- **Load Balancer**: $20/month
- **Total**: **~$720-1,320/month**

### Self-Hosted Kubernetes
- **Nodes**: Variable (depends on provider)
- **Storage**: $50-100/month
- **Bandwidth**: $50-200/month
- **Total**: **$100-1,000/month** (highly variable)

---

**Choose your deployment path and follow the detailed guide! 🚀**

