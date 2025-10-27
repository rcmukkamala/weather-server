# Google Cloud Platform (GCP) Deployment Guide

Complete guide for deploying Weather Server to GCP using managed services.

## Architecture on GCP

```
┌─────────────────────────────────────────────────────────────┐
│                    Google Cloud Platform                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Google Kubernetes Engine (GKE)           │  │
│  │                                                        │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │  Server  │  │Aggregator│  │ Alarming │           │  │
│  │  │   Pods   │  │   Pod    │  │   Pods   │           │  │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘           │  │
│  │       │             │              │                  │  │
│  └───────┼─────────────┼──────────────┼──────────────────┘  │
│          │             │              │                     │
│          ▼             ▼              ▼                     │
│  ┌────────────┐  ┌──────────┐  ┌────────────┐            │
│  │   Cloud    │  │  Cloud   │  │ Memorystore│            │
│  │   Pub/Sub  │  │   SQL    │  │   Redis    │            │
│  │  (Kafka)   │  │(Postgres)│  │            │            │
│  └────────────┘  └──────────┘  └────────────┘            │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐ │
│  │        Cloud Load Balancer (TCP Server Ingress)       │ │
│  └───────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Stackdriver │  │  Secret Mgr  │  │ Cloud Build  │    │
│  │   Monitoring  │  │              │  │   (CI/CD)    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- GCP Account with billing enabled
- gcloud CLI installed and configured
- kubectl installed
- Docker installed

## Step-by-Step Deployment

### 1. Initial Setup

```bash
# Set variables
export PROJECT_ID="your-project-id"
export REGION="us-central1"
export ZONE="${REGION}-a"
export CLUSTER_NAME="weather-cluster"
export CLUSTER_VERSION="1.28"

# Set project
gcloud config set project ${PROJECT_ID}

# Enable required APIs
gcloud services enable \
  container.googleapis.com \
  compute.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  monitoring.googleapis.com \
  logging.googleapis.com
```

### 2. Create GKE Cluster

```bash
# Create GKE cluster with recommended configuration
gcloud container clusters create ${CLUSTER_NAME} \
  --region ${REGION} \
  --cluster-version ${CLUSTER_VERSION} \
  --machine-type e2-standard-4 \
  --num-nodes 2 \
  --min-nodes 2 \
  --max-nodes 10 \
  --enable-autoscaling \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-ip-alias \
  --enable-stackdriver-kubernetes \
  --addons HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver \
  --workload-pool=${PROJECT_ID}.svc.id.goog \
  --enable-shielded-nodes \
  --shielded-secure-boot \
  --shielded-integrity-monitoring \
  --no-enable-basic-auth \
  --no-issue-client-certificate

# Get credentials
gcloud container clusters get-credentials ${CLUSTER_NAME} --region ${REGION}

# Verify connection
kubectl cluster-info
```

### 3. Create Cloud SQL (PostgreSQL)

```bash
# Create Cloud SQL instance
gcloud sql instances create weather-postgres \
  --database-version=POSTGRES_15 \
  --tier=db-custom-2-8192 \
  --region=${REGION} \
  --network=default \
  --enable-bin-log \
  --backup-start-time=03:00 \
  --maintenance-window-day=SUN \
  --maintenance-window-hour=04 \
  --maintenance-release-channel=production

# Create database
gcloud sql databases create weather_db --instance=weather-postgres

# Create user
gcloud sql users create weather_user \
  --instance=weather-postgres \
  --password=$(openssl rand -base64 32)

# Get connection name
gcloud sql instances describe weather-postgres --format="value(connectionName)"
# Output: project-id:region:instance-name

# Get IP address
gcloud sql instances describe weather-postgres --format="value(ipAddresses[0].ipAddress)"
```

#### Enable Cloud SQL Proxy (for GKE access)

```bash
# The Weather Server will connect via private IP
# Ensure your GKE cluster is in the same VPC

# Or use Cloud SQL Proxy sidecar (recommended)
# See k8s manifests below for sidecar configuration
```

### 4. Create Memorystore (Redis)

```bash
# Create Redis instance
gcloud redis instances create weather-redis \
  --size=5 \
  --region=${REGION} \
  --redis-version=redis_7_0 \
  --tier=standard \
  --enable-auth

# Get host and auth string
gcloud redis instances describe weather-redis --region=${REGION} --format="value(host)"
gcloud redis instances describe weather-redis --region=${REGION} --format="value(authString)"
```

### 5. Setup Kafka (Cloud Pub/Sub or Confluent Cloud)

#### Option A: Confluent Cloud (Recommended)

```bash
# Sign up at https://confluent.cloud
# Create cluster
# Get bootstrap servers and API keys
# Update ConfigMap with bootstrap servers
```

#### Option B: Self-hosted Kafka on GKE

```bash
# Use Strimzi operator
kubectl create namespace kafka
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Apply Kafka cluster manifest (see kafka-gke.yaml below)
kubectl apply -f deploy/gcp/kafka-gke.yaml -n kafka
```

#### Option C: Cloud Pub/Sub (with Kafka adapter)

```bash
# Create topics
gcloud pubsub topics create weather-metrics-raw
gcloud pubsub topics create weather-alarms

# Create subscriptions
gcloud pubsub subscriptions create db-writer-sub --topic=weather-metrics-raw
gcloud pubsub subscriptions create alarming-sub --topic=weather-metrics-raw
gcloud pubsub subscriptions create notification-sub --topic=weather-alarms

# Note: This requires code changes to use Pub/Sub SDK instead of Kafka
```

### 6. Setup Container Registry

```bash
# Create Artifact Registry repository
gcloud artifacts repositories create weather-server \
  --repository-format=docker \
  --location=${REGION} \
  --description="Weather Server container images"

# Configure Docker auth
gcloud auth configure-docker ${REGION}-docker.pkg.dev

# Set registry variable
export REGISTRY="${REGION}-docker.pkg.dev/${PROJECT_ID}/weather-server"
```

### 7. Build and Push Images

```bash
# Build and push all images
export VERSION="v1.0.0"

docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:${VERSION} .
docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:${VERSION} .
docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:${VERSION} .
docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:${VERSION} .

docker push ${REGISTRY}/weather-server:${VERSION}
docker push ${REGISTRY}/weather-aggregator:${VERSION}
docker push ${REGISTRY}/weather-alarming:${VERSION}
docker push ${REGISTRY}/weather-notification:${VERSION}

# Tag as latest
docker tag ${REGISTRY}/weather-server:${VERSION} ${REGISTRY}/weather-server:latest
docker tag ${REGISTRY}/weather-aggregator:${VERSION} ${REGISTRY}/weather-aggregator:latest
docker tag ${REGISTRY}/weather-alarming:${VERSION} ${REGISTRY}/weather-alarming:latest
docker tag ${REGISTRY}/weather-notification:${VERSION} ${REGISTRY}/weather-notification:latest

docker push ${REGISTRY}/weather-server:latest
docker push ${REGISTRY}/weather-aggregator:latest
docker push ${REGISTRY}/weather-alarming:latest
docker push ${REGISTRY}/weather-notification:latest
```

### 8. Setup Secret Manager

```bash
# Create secrets
echo -n "your_db_password" | gcloud secrets create db-password --data-file=-
echo -n "your_smtp_password" | gcloud secrets create smtp-password --data-file=-
echo -n "weather_user" | gcloud secrets create db-user --data-file=-
echo -n "your-email@gmail.com" | gcloud secrets create smtp-username --data-file=-

# Grant GKE access
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${PROJECT_ID}.svc.id.goog[weather-system/default]" \
  --role="roles/secretmanager.secretAccessor"
```

### 9. Create Kubernetes Secrets

```bash
# Create namespace
kubectl create namespace weather-system

# Create secret from Secret Manager (using Workload Identity)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: weather-secrets
  namespace: weather-system
type: Opaque
stringData:
  DB_USER: "weather_user"
  DB_PASSWORD: "$(gcloud secrets versions access latest --secret=db-password)"
  REDIS_PASSWORD: "$(gcloud redis instances describe weather-redis --region=${REGION} --format='value(authString)')"
  SMTP_USERNAME: "$(gcloud secrets versions access latest --secret=smtp-username)"
  SMTP_PASSWORD: "$(gcloud secrets versions access latest --secret=smtp-password)"
EOF
```

### 10. Update ConfigMap

```bash
# Get service endpoints
CLOUDSQL_IP=$(gcloud sql instances describe weather-postgres --format="value(ipAddresses[0].ipAddress)")
REDIS_IP=$(gcloud redis instances describe weather-redis --region=${REGION} --format="value(host)")
KAFKA_BROKERS="your-kafka-bootstrap-servers"  # From Confluent Cloud or self-hosted

# Create ConfigMap
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: weather-config
  namespace: weather-system
data:
  DB_HOST: "${CLOUDSQL_IP}"
  DB_PORT: "5432"
  DB_NAME: "weather_db"
  DB_SSLMODE: "require"
  REDIS_ADDR: "${REDIS_IP}:6379"
  REDIS_DB: "0"
  KAFKA_BROKERS: "${KAFKA_BROKERS}"
  KAFKA_TOPIC_METRICS: "weather.metrics.raw"
  KAFKA_TOPIC_ALARMS: "weather.alarms"
  KAFKA_NUM_PARTITIONS: "10"
  TCP_PORT: "8080"
  TCP_MAX_CONNECTIONS: "10000"
  TCP_IDENTIFY_TIMEOUT: "10s"
  TCP_INACTIVITY_TIMEOUT: "2m"
  AGGREGATION_HOURLY_DELAY: "5m"
  AGGREGATION_DAILY_TIME: "00:05"
  SMTP_HOST: "smtp.gmail.com"
  SMTP_PORT: "587"
  SMTP_FROM: "weather-server@${PROJECT_ID}.iam.gserviceaccount.com"
  SMTP_TO: "admin@example.com"
EOF
```

### 11. Deploy Applications

```bash
# Update image references in deployment files
find k8s/ -name "*-deployment.yaml" -exec sed -i "s|gcr.io/YOUR_PROJECT_ID|${REGISTRY}|g" {} +

# Deploy (skip postgres, redis, kafka as we're using managed services)
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/aggregator-deployment.yaml
kubectl apply -f k8s/alarming-deployment.yaml
kubectl apply -f k8s/notification-deployment.yaml

# Check status
kubectl get pods -n weather-system -w
```

### 12. Run Database Migrations

```bash
# Port-forward to a server pod
kubectl port-forward -n weather-system deployment/weather-server 5432:5432

# Or create a migration job
kubectl create job --from=cronjob/db-migration migrate-initial -n weather-system
```

### 13. Setup Load Balancer

```bash
# The Service with type LoadBalancer will automatically create a GCP Load Balancer
kubectl get svc weather-server-service -n weather-system

# Get external IP
EXTERNAL_IP=$(kubectl get svc weather-server-service -n weather-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Weather Server TCP endpoint: ${EXTERNAL_IP}:8080"
```

## Production Enhancements

### 1. Cloud SQL Proxy Sidecar

Update server deployment with Cloud SQL Proxy:

```yaml
- name: cloud-sql-proxy
  image: gcr.io/cloudsql-docker/gce-proxy:latest
  command:
    - "/cloud_sql_proxy"
    - "-instances=PROJECT_ID:REGION:INSTANCE=tcp:5432"
  securityContext:
    runAsNonRoot: true
```

### 2. Workload Identity

```bash
# Create service account
gcloud iam service-accounts create weather-gke-sa \
  --display-name="Weather Server GKE SA"

# Bind to K8s service account
gcloud iam service-accounts add-iam-policy-binding \
  weather-gke-sa@${PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[weather-system/default]"

# Annotate K8s service account
kubectl annotate serviceaccount default \
  -n weather-system \
  iam.gke.io/gcp-service-account=weather-gke-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

### 3. Cloud Monitoring

```bash
# Metrics are automatically sent to Cloud Monitoring

# Create custom dashboard
gcloud monitoring dashboards create --config-from-file=deploy/gcp/dashboard.json
```

### 4. Cloud Logging

```bash
# View logs
gcloud logging read "resource.type=k8s_container AND resource.labels.namespace_name=weather-system" --limit 50

# Create log-based metrics
gcloud logging metrics create weather_errors \
  --description="Weather server errors" \
  --log-filter='resource.type="k8s_container" AND resource.labels.namespace_name="weather-system" AND severity>=ERROR'
```

### 5. Cloud Build CI/CD

See `deploy/gcp/cloudbuild.yaml` for automated build and deployment.

```bash
# Trigger build
gcloud builds submit --config=deploy/gcp/cloudbuild.yaml
```

## Cost Optimization

### 1. Committed Use Discounts
- GKE: Use committed use for sustained workloads (30-57% discount)
- Cloud SQL: Use committed use for database (up to 35% discount)

### 2. Preemptible Nodes
```bash
# Add preemptible node pool for non-critical workloads
gcloud container node-pools create preemptible-pool \
  --cluster=${CLUSTER_NAME} \
  --region=${REGION} \
  --preemptible \
  --machine-type=e2-standard-2 \
  --num-nodes=2
```

### 3. Autoscaling
- GKE Cluster Autoscaler (already enabled)
- Horizontal Pod Autoscaler (already configured)
- Vertical Pod Autoscaler (optional)

### 4. Budget Alerts
```bash
# Create budget alert
gcloud billing budgets create \
  --billing-account=$(gcloud beta billing projects describe ${PROJECT_ID} --format="value(billingAccountName)") \
  --display-name="Weather Server Budget" \
  --budget-amount=500USD \
  --threshold-rule=percent=50 \
  --threshold-rule=percent=90 \
  --threshold-rule=percent=100
```

## Estimated Monthly Costs

| Service | Configuration | Monthly Cost (USD) |
|---------|--------------|-------------------|
| GKE Cluster | 2-10 e2-standard-4 nodes | $150-750 |
| Cloud SQL | db-custom-2-8192, HA | $250 |
| Memorystore | Standard, 5GB | $130 |
| Kafka (Confluent) | Basic cluster | $300 |
| Load Balancer | Single IP | $20 |
| Cloud Storage | Backups | $10 |
| Networking | Egress (moderate) | $50 |
| **Total** | | **$910-1,510/month** |

## Maintenance

### Backups
```bash
# Cloud SQL automated backups (already enabled)
# Manual backup
gcloud sql backups create --instance=weather-postgres

# Restore from backup
gcloud sql backups restore BACKUP_ID --backup-instance=weather-postgres
```

### Updates
```bash
# Update GKE cluster
gcloud container clusters upgrade ${CLUSTER_NAME} --region=${REGION}

# Update applications
./deploy/gcp/deploy.sh v1.1.0
```

## Monitoring & Alerts

Access Cloud Console:
- **Metrics**: https://console.cloud.google.com/monitoring
- **Logs**: https://console.cloud.google.com/logs
- **Traces**: https://console.cloud.google.com/traces

## Troubleshooting

See [Troubleshooting Guide](./TROUBLESHOOTING.md)

## Next Steps

- [Setup Monitoring](../monitoring/README.md)
- [Configure Alerts](../alerts/README.md)
- [Disaster Recovery Plan](../dr/README.md)

