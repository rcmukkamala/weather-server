# ✅ Kubernetes Deployment Checklist - 2 Worker Nodes

Complete pre-deployment verification and deployment checklist for a 2-node Kubernetes cluster.

---

## 🎯 Test Results Summary

### ✅ All Tests Passed
```
✅ Connection Manager Tests: 7/7 PASSED
✅ Timer Manager Tests: 5/5 PASSED
✅ Build Verification: ALL 4 BINARIES COMPILED
✅ Code Quality: NO LINTER ERRORS
```

**Total Test Coverage:**
- `internal/connection`: 7 tests ✅
- `internal/timer`: 5 tests ✅
- **Build Status**: All services compile successfully ✅

---

## 📦 Pre-Deployment Verification

### ✅ Repository Status

| Item | Status | Details |
|------|--------|---------|
| **Go Code** | ✅ READY | 3,800+ lines, compiles successfully |
| **Unit Tests** | ✅ PASSED | 12/12 tests passing |
| **Docker Images** | ✅ READY | 4 Dockerfiles created |
| **K8s Manifests** | ✅ READY | 10 YAML files |
| **Documentation** | ✅ COMPLETE | 2,400+ lines |
| **Migrations** | ✅ READY | 2 SQL files (6 tables) |

---

## 🖥️ 2-Node Cluster Requirements

### Minimum Specifications

**Per Node:**
- **CPU**: 4 vCPUs
- **Memory**: 8 GB RAM
- **Storage**: 100 GB SSD
- **Network**: 1 Gbps

**Total Cluster:**
- **CPU**: 8 vCPUs (4 per node)
- **Memory**: 16 GB RAM (8 per node)
- **Storage**: 200 GB (100 per node)

### Resource Allocation

```
Node 1 (Master + Worker):
├── Kubernetes System Pods: ~1 CPU, ~2 GB RAM
├── Weather Server: 1-2 replicas
├── PostgreSQL: 1 replica
└── Kafka: 1-2 brokers

Node 2 (Worker):
├── Aggregator: 1 replica
├── Alarming: 1-2 replicas
├── Notification: 1 replica
├── Redis: 1 replica
└── Kafka: 1-2 brokers
```

---

## 📋 DEPLOYMENT CHECKLIST

### Phase 1: Pre-Deployment Checks

#### ✅ 1.1 Infrastructure Ready

- [ ] **Kubernetes cluster running** (1.24+)
  ```bash
  kubectl version --short
  # Client: v1.24+
  # Server: v1.24+
  ```

- [ ] **2 nodes in Ready state**
  ```bash
  kubectl get nodes
  # Should show 2 nodes, both STATUS: Ready
  ```

- [ ] **Node resources verified**
  ```bash
  kubectl top nodes
  # Verify CPU and Memory availability
  ```

- [ ] **kubectl configured**
  ```bash
  kubectl cluster-info
  # Should show cluster endpoint
  ```

- [ ] **Storage class available**
  ```bash
  kubectl get storageclass
  # At least 1 storage class available
  ```

#### ✅ 1.2 Container Registry Access

- [ ] **Registry chosen** (Docker Hub, GCR, ECR, etc.)
  ```bash
  export REGISTRY="your-registry.io/weather-server"
  echo $REGISTRY
  ```

- [ ] **Docker logged in**
  ```bash
  docker login $REGISTRY
  # Login Succeeded
  ```

- [ ] **Registry accessible from cluster**
  ```bash
  # Test pull access
  kubectl run test --image=${REGISTRY}/test:latest --dry-run=client
  ```

#### ✅ 1.3 Configuration Files Ready

- [ ] **Secrets edited** (`k8s/secrets.yaml`)
  ```bash
  # Verify DB_PASSWORD is set
  grep "CHANGE_ME" k8s/secrets.yaml
  # Should return NO results
  ```

- [ ] **ConfigMap reviewed** (`k8s/configmap.yaml`)
  ```bash
  # Verify endpoints are correct
  cat k8s/configmap.yaml | grep "DB_HOST\|REDIS_ADDR\|KAFKA_BROKERS"
  ```

- [ ] **Registry paths updated**
  ```bash
  # Verify image references
  grep "image:" k8s/*-deployment.yaml
  # Should show YOUR registry, not gcr.io/YOUR_PROJECT_ID
  ```

---

### Phase 2: Build and Push Images

#### ✅ 2.1 Build Docker Images

- [ ] **Set registry variable**
  ```bash
  export REGISTRY="your-registry.io/weather-server"
  export VERSION="v1.0.0"
  ```

- [ ] **Build server image**
  ```bash
  docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:${VERSION} .
  # ✓ Successfully tagged
  ```

- [ ] **Build aggregator image**
  ```bash
  docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:${VERSION} .
  # ✓ Successfully tagged
  ```

- [ ] **Build alarming image**
  ```bash
  docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:${VERSION} .
  # ✓ Successfully tagged
  ```

- [ ] **Build notification image**
  ```bash
  docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:${VERSION} .
  # ✓ Successfully tagged
  ```

- [ ] **Tag as latest**
  ```bash
  docker tag ${REGISTRY}/weather-server:${VERSION} ${REGISTRY}/weather-server:latest
  docker tag ${REGISTRY}/weather-aggregator:${VERSION} ${REGISTRY}/weather-aggregator:latest
  docker tag ${REGISTRY}/weather-alarming:${VERSION} ${REGISTRY}/weather-alarming:latest
  docker tag ${REGISTRY}/weather-notification:${VERSION} ${REGISTRY}/weather-notification:latest
  ```

#### ✅ 2.2 Push Images

- [ ] **Push server**
  ```bash
  docker push ${REGISTRY}/weather-server:${VERSION}
  docker push ${REGISTRY}/weather-server:latest
  ```

- [ ] **Push aggregator**
  ```bash
  docker push ${REGISTRY}/weather-aggregator:${VERSION}
  docker push ${REGISTRY}/weather-aggregator:latest
  ```

- [ ] **Push alarming**
  ```bash
  docker push ${REGISTRY}/weather-alarming:${VERSION}
  docker push ${REGISTRY}/weather-alarming:latest
  ```

- [ ] **Push notification**
  ```bash
  docker push ${REGISTRY}/weather-notification:${VERSION}
  docker push ${REGISTRY}/weather-notification:latest
  ```

- [ ] **Verify images in registry**
  ```bash
  docker images | grep weather
  # Should show all 4 images
  ```

---

### Phase 3: Deploy Infrastructure (Database, Redis, Kafka)

#### ✅ 3.1 Create Namespace

- [ ] **Apply namespace**
  ```bash
  kubectl apply -f k8s/namespace.yaml
  # namespace/weather-system created
  ```

- [ ] **Verify namespace**
  ```bash
  kubectl get namespace weather-system
  # STATUS: Active
  ```

#### ✅ 3.2 Deploy Secrets and Config

- [ ] **Apply secrets** (MUST EDIT FIRST!)
  ```bash
  kubectl apply -f k8s/secrets.yaml
  # secret/weather-secrets created
  ```

- [ ] **Verify secrets**
  ```bash
  kubectl get secrets -n weather-system
  # weather-secrets should be listed
  ```

- [ ] **Apply configmap**
  ```bash
  kubectl apply -f k8s/configmap.yaml
  # configmap/weather-config created
  ```

#### ✅ 3.3 Deploy PostgreSQL

- [ ] **Apply PostgreSQL StatefulSet**
  ```bash
  kubectl apply -f k8s/postgres.yaml
  # service/postgres-service created
  # statefulset.apps/postgres created
  ```

- [ ] **Wait for PostgreSQL to be ready**
  ```bash
  kubectl wait --for=condition=ready pod -l app=postgres -n weather-system --timeout=300s
  # pod/postgres-0 condition met
  ```

- [ ] **Verify PostgreSQL running**
  ```bash
  kubectl get pods -n weather-system -l app=postgres
  # STATUS: Running, READY: 1/1
  ```

- [ ] **Check PostgreSQL logs**
  ```bash
  kubectl logs -n weather-system statefulset/postgres --tail=20
  # Should show "database system is ready to accept connections"
  ```

#### ✅ 3.4 Deploy Redis

- [ ] **Apply Redis StatefulSet**
  ```bash
  kubectl apply -f k8s/redis.yaml
  # service/redis-service created
  # statefulset.apps/redis created
  ```

- [ ] **Wait for Redis to be ready**
  ```bash
  kubectl wait --for=condition=ready pod -l app=redis -n weather-system --timeout=300s
  ```

- [ ] **Verify Redis running**
  ```bash
  kubectl get pods -n weather-system -l app=redis
  # STATUS: Running, READY: 1/1
  ```

#### ✅ 3.5 Deploy Kafka (Adjusted for 2 Nodes)

**⚠️ IMPORTANT: Edit `k8s/kafka.yaml` for 2-node cluster**

- [ ] **Reduce Kafka replicas** (edit k8s/kafka.yaml)
  ```yaml
  # Line 58: Change from 3 to 2 replicas
  replicas: 2  # Changed from 3
  
  # Line 77: Change replication factor
  KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "2"  # Changed from 3
  KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "2"  # Changed from 3
  ```

- [ ] **Apply Kafka**
  ```bash
  kubectl apply -f k8s/kafka.yaml
  # Created zookeeper and kafka resources
  ```

- [ ] **Wait for Zookeeper**
  ```bash
  kubectl wait --for=condition=ready pod -l app=zookeeper -n weather-system --timeout=300s
  ```

- [ ] **Wait for Kafka (2 brokers)**
  ```bash
  kubectl wait --for=condition=ready pod -l app=kafka -n weather-system --timeout=600s
  # This may take 5-10 minutes
  ```

- [ ] **Verify Kafka running**
  ```bash
  kubectl get pods -n weather-system -l app=kafka
  # Should show kafka-0 and kafka-1 (2 pods)
  # Both STATUS: Running, READY: 1/1
  ```

- [ ] **Check Kafka logs**
  ```bash
  kubectl logs -n weather-system statefulset/kafka --tail=20
  # Should show "started (kafka.server.KafkaServer)"
  ```

#### ✅ 3.6 Verify Infrastructure

- [ ] **All infrastructure pods running**
  ```bash
  kubectl get pods -n weather-system
  # Expected: postgres-0, redis-0, kafka-0, kafka-1, zookeeper-0
  # All should be Running and Ready
  ```

- [ ] **Check services**
  ```bash
  kubectl get svc -n weather-system
  # postgres-service, redis-service, kafka-service, zookeeper-service
  ```

- [ ] **Check PVCs bound**
  ```bash
  kubectl get pvc -n weather-system
  # All should be STATUS: Bound
  ```

---

### Phase 4: Deploy Application Services

#### ✅ 4.1 Deploy TCP Server

- [ ] **Update image reference**
  ```bash
  sed -i "s|gcr.io/YOUR_PROJECT_ID|${REGISTRY}|g" k8s/server-deployment.yaml
  ```

- [ ] **Apply server deployment**
  ```bash
  kubectl apply -f k8s/server-deployment.yaml
  # deployment/weather-server created
  # service/weather-server-service created
  # hpa/weather-server-hpa created
  ```

- [ ] **Wait for server pods**
  ```bash
  kubectl wait --for=condition=available deployment/weather-server -n weather-system --timeout=300s
  ```

- [ ] **Verify server running**
  ```bash
  kubectl get pods -n weather-system -l app=weather-server
  # Should show 3 pods (may scale to 2 if resources limited)
  ```

- [ ] **Check server logs (migrations should run)**
  ```bash
  kubectl logs -n weather-system deployment/weather-server --tail=50
  # Look for:
  # "Running migration: 001_initial_schema.sql"
  # "Running migration: 002_alarm_tables.sql"
  # "✓ Weather Server is running"
  ```

#### ✅ 4.2 Deploy Aggregator

- [ ] **Apply aggregator deployment**
  ```bash
  kubectl apply -f k8s/aggregator-deployment.yaml
  # deployment/weather-aggregator created
  ```

- [ ] **Verify aggregator running**
  ```bash
  kubectl get pods -n weather-system -l app=weather-aggregator
  # 1 pod, STATUS: Running
  ```

- [ ] **Check aggregator logs**
  ```bash
  kubectl logs -n weather-system deployment/weather-aggregator --tail=20
  # "✓ Aggregation Service is running"
  # "Next hourly aggregation scheduled for: ..."
  ```

#### ✅ 4.3 Deploy Alarming

- [ ] **Apply alarming deployment**
  ```bash
  kubectl apply -f k8s/alarming-deployment.yaml
  # deployment/weather-alarming created
  # hpa/weather-alarming-hpa created
  ```

- [ ] **Verify alarming running**
  ```bash
  kubectl get pods -n weather-system -l app=weather-alarming
  # 2 pods, STATUS: Running
  ```

- [ ] **Check alarming logs**
  ```bash
  kubectl logs -n weather-system deployment/weather-alarming --tail=20
  # "✓ Alarming Service is running"
  ```

#### ✅ 4.4 Deploy Notification

- [ ] **Apply notification deployment**
  ```bash
  kubectl apply -f k8s/notification-deployment.yaml
  # deployment/weather-notification created
  ```

- [ ] **Verify notification running**
  ```bash
  kubectl get pods -n weather-system -l app=weather-notification
  # 2 pods, STATUS: Running
  ```

---

### Phase 5: Verification and Testing

#### ✅ 5.1 Overall Status Check

- [ ] **All pods running**
  ```bash
  kubectl get pods -n weather-system
  # Expected pods:
  # - weather-server-xxx (2-3 pods)
  # - weather-aggregator-xxx (1 pod)
  # - weather-alarming-xxx (2 pods)
  # - weather-notification-xxx (2 pods)
  # - postgres-0 (1 pod)
  # - redis-0 (1 pod)
  # - kafka-0, kafka-1 (2 pods)
  # - zookeeper-0 (1 pod)
  # Total: ~12-14 pods
  ```

- [ ] **No pods in CrashLoopBackOff**
  ```bash
  kubectl get pods -n weather-system | grep -i crash
  # Should return nothing
  ```

- [ ] **All services exist**
  ```bash
  kubectl get svc -n weather-system
  # weather-server-service (LoadBalancer)
  # postgres-service, redis-service, kafka-service, zookeeper-service
  ```

#### ✅ 5.2 Get External IP

- [ ] **Check LoadBalancer status**
  ```bash
  kubectl get svc weather-server-service -n weather-system
  # EXTERNAL-IP should show an IP (not <pending>)
  # On some clusters, this may stay <pending> if LoadBalancer isn't supported
  ```

- [ ] **If LoadBalancer pending, use NodePort**
  ```bash
  # If EXTERNAL-IP is <pending>:
  kubectl patch svc weather-server-service -n weather-system -p '{"spec":{"type":"NodePort"}}'
  kubectl get svc weather-server-service -n weather-system
  # Note the NodePort (e.g., 30123)
  # Access via: <NODE_IP>:30123
  ```

- [ ] **Or use port-forward for testing**
  ```bash
  kubectl port-forward -n weather-system svc/weather-server-service 8080:8080
  # Access via: localhost:8080
  ```

#### ✅ 5.3 Test TCP Connection

- [ ] **Test with netcat**
  ```bash
  nc -zv <EXTERNAL_IP> 8080
  # Or: nc -zv localhost 8080 (if using port-forward)
  # Connection succeeded
  ```

- [ ] **Run sample client**
  ```bash
  # In a new terminal:
  go run examples/client/main.go
  # Should connect and send metrics
  ```

- [ ] **Verify client logs**
  ```
  ✓ Connected to server
  → Sent identify message
  ← Received ack: identified
  → Sent metrics: temp=25.3°C, humidity=62.5%, wind=15.2 mph NW
  ```

#### ✅ 5.4 Verify Database

- [ ] **Check database connections**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- psql -U weather_user -d weather_db -c "\dt"
  # Should list 6 tables:
  # locations, raw_metrics, hourly_metrics, daily_summary, 
  # alarm_thresholds, alarms_log
  ```

- [ ] **Check locations table**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- \
    psql -U weather_user -d weather_db -c "SELECT COUNT(*) FROM locations;"
  # Should show > 0 after client connects
  ```

- [ ] **Check metrics ingestion**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- \
    psql -U weather_user -d weather_db -c "SELECT COUNT(*) FROM raw_metrics;"
  # Should increase over time as client sends data
  ```

#### ✅ 5.5 Verify Redis

- [ ] **Test Redis connection**
  ```bash
  kubectl exec -n weather-system statefulset/redis -- redis-cli ping
  # PONG
  ```

- [ ] **Check alarm states (if any thresholds configured)**
  ```bash
  kubectl exec -n weather-system statefulset/redis -- redis-cli KEYS "alarm_state:*"
  # Will show alarm states if any exist
  ```

#### ✅ 5.6 Check Resource Usage

- [ ] **Node resource usage**
  ```bash
  kubectl top nodes
  # Verify nodes aren't overloaded (< 80% CPU/Memory)
  ```

- [ ] **Pod resource usage**
  ```bash
  kubectl top pods -n weather-system
  # Check if any pods are hitting limits
  ```

- [ ] **HPA status**
  ```bash
  kubectl get hpa -n weather-system
  # Check auto-scaling status
  ```

---

### Phase 6: Post-Deployment Configuration

#### ✅ 6.1 Setup Alarm Thresholds (Optional)

- [ ] **Add sample alarm thresholds**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- \
    psql -U weather_user -d weather_db -f /dev/stdin < scripts/setup_alarms.sql
  ```

- [ ] **Verify thresholds**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- \
    psql -U weather_user -d weather_db -c "SELECT * FROM alarm_thresholds;"
  ```

#### ✅ 6.2 Configure SMTP (Optional)

- [ ] **Update secrets with SMTP credentials**
  ```bash
  kubectl edit secret weather-secrets -n weather-system
  # Add SMTP_USERNAME and SMTP_PASSWORD
  ```

- [ ] **Restart notification service**
  ```bash
  kubectl rollout restart deployment/weather-notification -n weather-system
  ```

---

### Phase 7: Monitoring and Maintenance

#### ✅ 7.1 Setup Monitoring

- [ ] **Enable metrics-server (if not already)**
  ```bash
  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
  ```

- [ ] **Verify metrics available**
  ```bash
  kubectl top nodes
  kubectl top pods -n weather-system
  ```

#### ✅ 7.2 Setup Log Aggregation

- [ ] **View logs for debugging**
  ```bash
  # Server logs
  kubectl logs -f -n weather-system deployment/weather-server
  
  # Aggregator logs
  kubectl logs -f -n weather-system deployment/weather-aggregator
  
  # Alarming logs
  kubectl logs -f -n weather-system deployment/weather-alarming
  ```

#### ✅ 7.3 Backup Strategy

- [ ] **Export PostgreSQL data**
  ```bash
  kubectl exec -n weather-system statefulset/postgres -- \
    pg_dump -U weather_user weather_db > backup-$(date +%Y%m%d).sql
  ```

- [ ] **Schedule regular backups**
  ```bash
  # Consider using Velero or database-specific backup tools
  ```

---

## 🚨 Troubleshooting Common Issues

### Issue 1: Pods in Pending State

**Symptom:** Pods stuck in Pending
```bash
kubectl get pods -n weather-system
# STATUS: Pending
```

**Solutions:**
```bash
# Check node resources
kubectl describe pod <pod-name> -n weather-system
# Look for "Insufficient cpu" or "Insufficient memory"

# Solution A: Scale down replicas
kubectl scale deployment weather-server --replicas=2 -n weather-system

# Solution B: Adjust resource requests in deployment YAML
# Edit k8s/server-deployment.yaml, reduce CPU/memory requests
```

### Issue 2: Kafka Pods Not Starting (2-Node Specific)

**Symptom:** Kafka pods crash with replication errors

**Solution:**
```bash
# Edit k8s/kafka.yaml
# Reduce replication factors from 3 to 2:
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "2"
KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "2"

# Reapply
kubectl delete -f k8s/kafka.yaml
kubectl apply -f k8s/kafka.yaml
```

### Issue 3: LoadBalancer Pending

**Symptom:** EXTERNAL-IP shows <pending>

**Solution:**
```bash
# Use NodePort instead
kubectl patch svc weather-server-service -n weather-system \
  -p '{"spec":{"type":"NodePort"}}'

# Get NodePort
kubectl get svc weather-server-service -n weather-system
# Access via: <NODE_IP>:<NODEPORT>
```

### Issue 4: Database Connection Failures

**Symptom:** Server logs show "Failed to connect to database"

**Solutions:**
```bash
# Check PostgreSQL running
kubectl get pods -n weather-system -l app=postgres

# Check service endpoints
kubectl get endpoints postgres-service -n weather-system

# Verify secrets
kubectl get secret weather-secrets -n weather-system -o yaml

# Test connection from server pod
kubectl exec -n weather-system deployment/weather-server -- \
  nc -zv postgres-service 5432
```

---

## 📊 Expected Resource Usage (2-Node Cluster)

### Cluster-Wide

| Resource | Used | Available | Utilization |
|----------|------|-----------|-------------|
| **CPU** | 4-5 cores | 8 cores | 50-60% |
| **Memory** | 8-10 GB | 16 GB | 50-60% |
| **Storage** | 30-50 GB | 200 GB | 15-25% |

### Per-Pod Resources

| Pod | CPU | Memory | Storage |
|-----|-----|--------|---------|
| weather-server | 250m | 256Mi | - |
| weather-aggregator | 100m | 128Mi | - |
| weather-alarming | 200m | 256Mi | - |
| weather-notification | 100m | 128Mi | - |
| postgres | 500m | 1Gi | 50Gi |
| redis | 100m | 256Mi | 10Gi |
| kafka (each) | 1000m | 2Gi | 50Gi |
| zookeeper | 500m | 512Mi | 10Gi |

---

## ✅ Final Verification Checklist

After completing all steps above:

- [ ] All 12-14 pods are Running
- [ ] No pods in CrashLoopBackOff or Error
- [ ] LoadBalancer or NodePort accessible
- [ ] Sample client successfully connects
- [ ] Database contains metrics (raw_metrics table)
- [ ] All 6 database tables created
- [ ] Logs show no errors
- [ ] Node CPU < 80%, Memory < 80%
- [ ] HPA functioning (if configured)
- [ ] Aggregation service scheduled jobs visible in logs

---

## 📞 Success Criteria

**Deployment is successful when:**

✅ **All pods running** (12-14 pods total)
✅ **TCP server accessible** via LoadBalancer/NodePort
✅ **Sample client connects** and sends data
✅ **Database receiving metrics** (SELECT COUNT(*) increases)
✅ **Migrations executed** (all 6 tables exist)
✅ **No errors in logs** for 5 minutes
✅ **Resources within limits** (< 80% utilization)

---

## 🎉 Post-Deployment

### Your Weather Server is now running on Kubernetes!

**Access your server:**
```bash
# Get endpoint
EXTERNAL_IP=$(kubectl get svc weather-server-service -n weather-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Weather Server TCP endpoint: ${EXTERNAL_IP}:8080"

# Or with NodePort:
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}')
NODE_PORT=$(kubectl get svc weather-server-service -n weather-system -o jsonpath='{.spec.ports[0].nodePort}')
echo "Weather Server TCP endpoint: ${NODE_IP}:${NODE_PORT}"
```

**Next steps:**
1. Configure alarm thresholds
2. Setup SMTP for email notifications
3. Monitor resource usage
4. Plan scaling strategy
5. Setup backup automation

---

**Total Deployment Time:** 30-60 minutes
**Difficulty:** Medium
**Recommended for:** Staging, Testing, Small Production

🚀 **Happy Deploying!**

