# Deployment Comparison: 2-Node vs 3-Node Kubernetes Cluster

Quick reference guide to help you choose between 2-node and 3-node deployment.

---

## 🎯 Quick Decision Matrix

| Your Scenario | Recommended Setup | Checklist File |
|--------------|-------------------|----------------|
| **Production environment** | **3-Node** ✅ | `DEPLOYMENT_CHECKLIST_3NODE.md` |
| **High availability required** | **3-Node** ✅ | `DEPLOYMENT_CHECKLIST_3NODE.md` |
| **Budget constrained** | 2-Node | `DEPLOYMENT_CHECKLIST_2NODE.md` |
| **Development/Testing** | 2-Node | `DEPLOYMENT_CHECKLIST_2NODE.md` |
| **Staging environment** | 3-Node recommended | `DEPLOYMENT_CHECKLIST_3NODE.md` |

---

## 📊 Detailed Comparison

### Infrastructure Requirements

| Aspect | 2-Node Cluster | 3-Node Cluster |
|--------|----------------|----------------|
| **Nodes** | 2 worker nodes | 3 worker nodes |
| **CPU per Node** | 4 vCPUs | 4 vCPUs |
| **Memory per Node** | 8 GB RAM | 8 GB RAM |
| **Storage per Node** | 100 GB SSD | 100 GB SSD |
| **Total CPU** | 8 vCPUs | **12 vCPUs** |
| **Total Memory** | 16 GB RAM | **24 GB RAM** |
| **Total Storage** | 200 GB | **300 GB** |
| **Monthly Cost (AWS)** | ~$120-200 | ~$180-300 |

---

### Kafka Configuration

| Feature | 2-Node | 3-Node |
|---------|--------|--------|
| **Brokers** | 2 (kafka-0, kafka-1) | **3 (kafka-0, kafka-1, kafka-2)** ✅ |
| **Replication Factor** | 2 (MUST edit YAML) ⚠️ | **3 (default, no edit)** ✅ |
| **Min In-Sync Replicas** | 1 | **2** ✅ |
| **Fault Tolerance** | 1 broker failure | **1 broker failure** ✅ |
| **Performance** | Good | **Excellent** ✅ |
| **Data Safety** | Good | **Best** ✅ |
| **Configuration Changes** | **REQUIRED** ⚠️ | **NONE** ✅ |

**Critical Difference:**
- **2-Node**: You MUST edit `k8s/kafka.yaml` before deployment
- **3-Node**: Deploy as-is, no edits needed

---

### High Availability

| Feature | 2-Node | 3-Node |
|---------|--------|--------|
| **Quorum** | ❌ No (even number) | ✅ **Yes (odd number)** |
| **Split-Brain Protection** | ⚠️ Limited | ✅ **Full** |
| **Node Failure Tolerance** | 1 node (50% capacity) | **1 node (66% capacity)** ✅ |
| **Service Continuity** | ⚠️ May degrade | ✅ **Seamless** |
| **Kafka Cluster Health** | ⚠️ 1 broker left if 1 fails | ✅ **2 brokers left if 1 fails** |
| **Production Ready** | Staging only | ✅ **Yes** |

---

### Application Scaling

| Service | 2-Node Replicas | 3-Node Replicas | HPA Max |
|---------|-----------------|-----------------|---------|
| **TCP Server** | 2-4 | **3-6** | 10 |
| **Aggregator** | 1 | 1 | - |
| **Alarming** | 2 | **2-3** | 5 |
| **Notification** | 2 | **2-3** | - |
| **PostgreSQL** | 1 | 1 | - |
| **Redis** | 1 | 1 | - |
| **Kafka** | 2 | **3** ✅ | - |
| **Zookeeper** | 1 | 1 | - |

---

### Resource Utilization

#### 2-Node Cluster

```
Total Capacity: 8 vCPU, 16 GB RAM
Used by System: ~2 vCPU, ~4 GB RAM
Available: ~6 vCPU, ~12 GB RAM

Utilization: 70-80% (tight)
Headroom: Limited
```

#### 3-Node Cluster

```
Total Capacity: 12 vCPU, 24 GB RAM
Used by System: ~2 vCPU, ~4 GB RAM
Available: ~10 vCPU, ~20 GB RAM

Utilization: 50-60% (comfortable)
Headroom: 40-50% for growth
```

---

### Deployment Complexity

| Step | 2-Node | 3-Node |
|------|--------|--------|
| **Infrastructure Setup** | Easier (fewer nodes) | Slightly more complex |
| **Kafka YAML Editing** | **REQUIRED** ⚠️ | **NOT REQUIRED** ✅ |
| **Pod Distribution** | Uneven | **Even** ✅ |
| **Testing** | Simple | More thorough |
| **Troubleshooting** | Easier | Standard |
| **Time to Deploy** | 25-45 min | 30-60 min |

---

### Cost Analysis

#### Example: AWS EC2 (us-east-1)

| Instance Type | vCPU | RAM | Storage | Price/Hour | Monthly (730h) |
|---------------|------|-----|---------|------------|----------------|
| **t3.xlarge** | 4 | 16GB | - | $0.1664 | $121.47 |
| **EBS gp3** | - | - | 100GB | $0.08/GB/mo | $8.00 |

**2-Node Setup:**
- 2 × t3.xlarge: $242.94/month
- 2 × 100GB EBS: $16.00/month
- **Total: ~$260/month**

**3-Node Setup:**
- 3 × t3.xlarge: $364.41/month
- 3 × 100GB EBS: $24.00/month
- **Total: ~$390/month**

**Cost Difference: $130/month (50% more for 50% more capacity)**

---

### Decision Flowchart

```
Do you need production-grade reliability?
│
├─ YES → Go with 3-Node ✅
│        ($390/mo, full HA, no Kafka edits)
│
└─ NO → Are you budget constrained?
         │
         ├─ YES → 2-Node (with Kafka edits)
         │        ($260/mo, staging only)
         │
         └─ NO → Still go with 3-Node ✅
                  (Future-proof, easier to manage)
```

---

## 🚨 Critical Differences

### What Changes Between 2-Node and 3-Node?

#### 1. Kafka Configuration (MOST IMPORTANT)

**2-Node - MUST Edit k8s/kafka.yaml:**
```yaml
# Line 58: Change replicas from 3 to 2
replicas: 2  # CHANGE THIS

# Lines 103-104: Change replication factors
- name: KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR
  value: "2"  # CHANGE THIS
- name: KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR
  value: "2"  # CHANGE THIS
```

**3-Node - No Changes Needed:**
```yaml
# Use default values, DO NOT EDIT
replicas: 3  # ✓ KEEP AS IS
KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "3"  # ✓ KEEP AS IS
KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "3"  # ✓ KEEP AS IS
```

#### 2. Application Replicas

**2-Node:**
```bash
# Conservative scaling
TCP Server: 2-4 replicas
Alarming: 2 replicas
Notification: 2 replicas
```

**3-Node:**
```bash
# Better distribution
TCP Server: 3-6 replicas (1-2 per node)
Alarming: 2-3 replicas
Notification: 2-3 replicas
```

#### 3. Resource Headroom

**2-Node:**
- CPU: ~25% headroom (tight)
- Memory: ~25% headroom (tight)
- Risk: May hit limits during traffic spikes

**3-Node:**
- CPU: ~40% headroom (comfortable)
- Memory: ~40% headroom (comfortable)
- Risk: Can handle traffic spikes gracefully

---

## 📋 Which Checklist to Use?

### Use `DEPLOYMENT_CHECKLIST_2NODE.md` if:

- [ ] Budget is limited (~$260/month)
- [ ] This is a dev/test/staging environment
- [ ] You don't need full high availability
- [ ] You're comfortable editing Kafka YAML
- [ ] You understand the HA limitations

### Use `DEPLOYMENT_CHECKLIST_3NODE.md` if:

- [ ] This is a production environment
- [ ] High availability is required
- [ ] You want the easiest deployment (no Kafka edits)
- [ ] You need room to scale
- [ ] Budget allows (~$390/month)
- [ ] **RECOMMENDED** ✅

---

## 🎓 Learning Path

### If You're New to Kubernetes:

**Start Here:** 2-Node Deployment
1. Read `DEPLOYMENT_CHECKLIST_2NODE.md`
2. Edit Kafka YAML (good learning experience)
3. Deploy and test
4. Understand resource constraints
5. **Then upgrade to 3-node for production**

### If You're Experienced:

**Go Straight to:** 3-Node Deployment
1. Read `DEPLOYMENT_CHECKLIST_3NODE.md`
2. Deploy as-is (no edits needed)
3. Enjoy production-grade setup immediately

---

## 🔄 Migrating from 2-Node to 3-Node

If you started with 2-node and want to upgrade:

```bash
# 1. Add third node to cluster
# (Cloud provider specific)

# 2. Revert Kafka YAML to defaults
git checkout k8s/kafka.yaml

# 3. Scale Kafka to 3 replicas
kubectl scale statefulset kafka --replicas=3 -n weather-system

# 4. Wait for third broker
kubectl wait --for=condition=ready pod kafka-2 -n weather-system --timeout=600s

# 5. Update topic replication
kubectl exec -n weather-system kafka-0 -- \
  kafka-configs --bootstrap-server localhost:9092 \
  --alter --entity-type topics --entity-name weather.metrics.raw \
  --add-config min.insync.replicas=2

# 6. Scale application services
kubectl scale deployment weather-server --replicas=3 -n weather-system
kubectl scale deployment weather-alarming --replicas=3 -n weather-system

# 7. Verify
kubectl get pods -n weather-system -o wide
```

---

## 📊 Real-World Scenarios

### Scenario 1: Startup MVP
- **Recommendation:** 2-Node
- **Reason:** Save costs while proving concept
- **Transition:** Move to 3-node when getting traction

### Scenario 2: Enterprise Application
- **Recommendation:** 3-Node (minimum)
- **Reason:** SLA requirements, uptime guarantees
- **Scale:** Consider 5+ nodes for large scale

### Scenario 3: IoT Weather Network
- **Recommendation:** 3-Node
- **Reason:** 24/7 data ingestion, can't afford downtime
- **Benefit:** Kafka handles replication perfectly

### Scenario 4: Research Project
- **Recommendation:** 2-Node
- **Reason:** Cost-effective, not mission-critical
- **Note:** Perfect for academic budgets

---

## ✅ Quick Reference Commands

### Check Your Current Setup

```bash
# How many nodes?
kubectl get nodes --no-headers | wc -l

# How many Kafka brokers?
kubectl get pods -n weather-system -l app=kafka --no-headers | wc -l

# What's the replication factor?
kubectl exec -n weather-system kafka-0 -- \
  kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic weather.metrics.raw | grep ReplicationFactor
```

### Switch Between Configurations

```bash
# For 2-Node: Edit Kafka
vi k8s/kafka.yaml
# Change replicas: 3 → 2
# Change replication factors: 3 → 2

# For 3-Node: Use defaults
git checkout k8s/kafka.yaml
```

---

## 🎯 Final Recommendation

### For Production: **3-Node Cluster** ✅

**Why?**
- Industry standard for HA
- No configuration changes needed
- 50% more capacity for 50% more cost (linear scaling)
- Future-proof
- Kafka works optimally with 3 replicas
- Can survive node failures gracefully

**Use:** `DEPLOYMENT_CHECKLIST_3NODE.md`

### For Dev/Staging: 2-Node is OK

**Why?**
- Cost savings
- Sufficient for testing
- Good learning experience

**Use:** `DEPLOYMENT_CHECKLIST_2NODE.md`

---

## 📚 Additional Resources

- `DEPLOYMENT_CHECKLIST_2NODE.md` - Detailed 2-node guide (150+ items)
- `DEPLOYMENT_CHECKLIST_3NODE.md` - Detailed 3-node guide (155+ items)
- `DEPLOYMENT.md` - Quick reference for all platforms
- `deploy/kubernetes/README.md` - Generic K8s deployment guide
- `deploy/gcp/README.md` - GCP-specific deployment guide

---

**Need Help Deciding?**

If you're still unsure, ask yourself:
1. Can I afford $130/month extra? → YES = 3-node
2. Do users depend on this 24/7? → YES = 3-node
3. Is this just for testing? → YES = 2-node
4. Do I want the easiest setup? → YES = 3-node

**When in doubt, go with 3-node.** ✅

