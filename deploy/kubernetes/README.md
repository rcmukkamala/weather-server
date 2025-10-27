# Kubernetes Deployment Guide

Complete guide for deploying Weather Server to any Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Docker registry access
- 4 CPUs, 8GB RAM minimum

## Quick Deploy

```bash
# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Create secrets (EDIT FIRST!)
kubectl apply -f k8s/secrets.yaml

# 3. Create config
kubectl apply -f k8s/configmap.yaml

# 4. Deploy infrastructure (or use managed services)
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/kafka.yaml

# 5. Wait for infrastructure
kubectl wait --for=condition=ready pod -l app=postgres -n weather-system --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n weather-system --timeout=300s
kubectl wait --for=condition=ready pod -l app=kafka -n weather-system --timeout=300s

# 6. Deploy applications
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/aggregator-deployment.yaml
kubectl apply -f k8s/alarming-deployment.yaml
kubectl apply -f k8s/notification-deployment.yaml

# 7. Check status
kubectl get pods -n weather-system
```

## Detailed Steps

### 1. Build and Push Docker Images

```bash
# Set your registry
export REGISTRY="gcr.io/YOUR_PROJECT_ID"
export VERSION="v1.0.0"

# Build all images
docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:${VERSION} .
docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:${VERSION} .
docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:${VERSION} .
docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:${VERSION} .

# Tag as latest
docker tag ${REGISTRY}/weather-server:${VERSION} ${REGISTRY}/weather-server:latest
docker tag ${REGISTRY}/weather-aggregator:${VERSION} ${REGISTRY}/weather-aggregator:latest
docker tag ${REGISTRY}/weather-alarming:${VERSION} ${REGISTRY}/weather-alarming:latest
docker tag ${REGISTRY}/weather-notification:${VERSION} ${REGISTRY}/weather-notification:latest

# Push to registry
docker push ${REGISTRY}/weather-server:${VERSION}
docker push ${REGISTRY}/weather-aggregator:${VERSION}
docker push ${REGISTRY}/weather-alarming:${VERSION}
docker push ${REGISTRY}/weather-notification:${VERSION}
docker push ${REGISTRY}/weather-server:latest
docker push ${REGISTRY}/weather-aggregator:latest
docker push ${REGISTRY}/weather-alarming:latest
docker push ${REGISTRY}/weather-notification:latest
```

### 2. Update Image References

Update all deployment YAML files with your registry:

```bash
# Replace placeholder with your actual registry
find k8s/ -name "*-deployment.yaml" -type f -exec sed -i '' 's|gcr.io/YOUR_PROJECT_ID|gcr.io/your-actual-project|g' {} +
```

### 3. Configure Secrets

**IMPORTANT**: Edit `k8s/secrets.yaml` with actual credentials:

```yaml
stringData:
  DB_PASSWORD: "YOUR_STRONG_PASSWORD_HERE"
  SMTP_USERNAME: "your-email@gmail.com"
  SMTP_PASSWORD: "your-app-password"
```

### 4. Deploy Infrastructure

#### Option A: Self-Hosted (Development)
```bash
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/kafka.yaml
```

#### Option B: Managed Services (Production - Recommended)

Update `k8s/configmap.yaml`:
```yaml
data:
  DB_HOST: "your-cloudsql-ip"
  REDIS_ADDR: "your-memorystore-ip:6379"
  KAFKA_BROKERS: "your-kafka-bootstrap-servers"
```

### 5. Run Database Migrations

```bash
# Port-forward to postgres
kubectl port-forward -n weather-system svc/postgres-service 5432:5432 &

# Run migrations locally
export DB_HOST=localhost
export DB_USER=weather_user
export DB_PASSWORD=your_password
export DB_NAME=weather_db

# Or exec into server pod
kubectl exec -it -n weather-system deployment/weather-server -- /bin/sh
# Inside pod: migrations are in /app/migrations
```

### 6. Verify Deployment

```bash
# Check all pods
kubectl get pods -n weather-system

# Check services
kubectl get svc -n weather-system

# Check logs
kubectl logs -f -n weather-system deployment/weather-server
kubectl logs -f -n weather-system deployment/weather-aggregator
kubectl logs -f -n weather-system deployment/weather-alarming
kubectl logs -f -n weather-system deployment/weather-notification

# Check HPA
kubectl get hpa -n weather-system
```

### 7. Get LoadBalancer IP

```bash
# Get external IP for TCP server
kubectl get svc weather-server-service -n weather-system

# Or use port-forward for testing
kubectl port-forward -n weather-system svc/weather-server-service 8080:8080
```

## Scaling

### Manual Scaling
```bash
# Scale TCP servers
kubectl scale deployment weather-server --replicas=5 -n weather-system

# Scale alarming service
kubectl scale deployment weather-alarming --replicas=3 -n weather-system
```

### Auto-scaling (HPA already configured)
```bash
# Check HPA status
kubectl get hpa -n weather-system

# Describe HPA
kubectl describe hpa weather-server-hpa -n weather-system
```

## Monitoring

### Logs
```bash
# Stream all logs
kubectl logs -f -l component=tcp-server -n weather-system
kubectl logs -f -l component=alarming -n weather-system

# Previous logs (if crashed)
kubectl logs --previous -n weather-system deployment/weather-server
```

### Metrics
```bash
# Top pods
kubectl top pods -n weather-system

# Describe pod for events
kubectl describe pod -n weather-system <pod-name>
```

### Health Checks
```bash
# Check pod status
kubectl get pods -n weather-system -o wide

# Check readiness
kubectl get pods -n weather-system -o json | jq '.items[].status.conditions'
```

## Troubleshooting

### Pods Not Starting
```bash
# Check events
kubectl get events -n weather-system --sort-by='.lastTimestamp'

# Describe deployment
kubectl describe deployment weather-server -n weather-system

# Check image pull
kubectl describe pod -n weather-system <pod-name> | grep -A 10 Events
```

### Database Connection Issues
```bash
# Test from pod
kubectl exec -it -n weather-system deployment/weather-server -- /bin/sh
# Inside: ping postgres-service
# Inside: nc -zv postgres-service 5432

# Check service endpoints
kubectl get endpoints -n weather-system
```

### Kafka Connection Issues
```bash
# Check Kafka logs
kubectl logs -n weather-system statefulset/kafka

# Test connection
kubectl exec -it -n weather-system deployment/weather-server -- /bin/sh
# Inside: nc -zv kafka-service 9092
```

## Cleanup

```bash
# Delete everything
kubectl delete namespace weather-system

# Or delete selectively
kubectl delete -f k8s/server-deployment.yaml
kubectl delete -f k8s/aggregator-deployment.yaml
kubectl delete -f k8s/alarming-deployment.yaml
kubectl delete -f k8s/notification-deployment.yaml
```

## Production Best Practices

### 1. Use Managed Services
- **Database**: Cloud SQL, RDS
- **Redis**: Memorystore, ElastiCache
- **Kafka**: Confluent Cloud, MSK, EventHub

### 2. Security
- Enable RBAC
- Use Pod Security Policies
- Network Policies
- Secrets Management (Vault, Sealed Secrets)
- Enable TLS for all services

### 3. High Availability
- Multi-zone deployment
- PodDisruptionBudgets
- Resource quotas
- Backup strategies

### 4. Monitoring
- Prometheus + Grafana
- ELK stack for logs
- Distributed tracing (Jaeger)
- Alertmanager

### 5. CI/CD
- GitOps (ArgoCD, Flux)
- Automated testing
- Canary deployments
- Rollback strategies

## Resource Requirements

| Component | Replicas | CPU (Request/Limit) | Memory (Request/Limit) |
|-----------|----------|---------------------|------------------------|
| TCP Server | 3-10 | 250m/500m | 256Mi/512Mi |
| Aggregator | 1 | 100m/200m | 128Mi/256Mi |
| Alarming | 2-5 | 200m/400m | 256Mi/512Mi |
| Notification | 2 | 100m/200m | 128Mi/256Mi |
| PostgreSQL | 1 | 500m/1000m | 1Gi/2Gi |
| Redis | 1 | 100m/200m | 256Mi/512Mi |
| Kafka | 3 | 1000m/2000m | 2Gi/4Gi |

**Total**: ~5-8 CPUs, ~8-12GB RAM

## Next Steps

- [GCP Deployment Guide](../gcp/README.md)
- [Monitoring Setup](../monitoring/README.md)
- [Backup & Disaster Recovery](../backup/README.md)

