#!/bin/bash
# Quick deployment script for GCP

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=${PROJECT_ID:-"your-project-id"}
REGION=${REGION:-"us-central1"}
CLUSTER_NAME=${CLUSTER_NAME:-"weather-cluster"}
VERSION=${1:-"latest"}

echo -e "${GREEN}Weather Server GCP Deployment${NC}"
echo "================================"
echo "Project: ${PROJECT_ID}"
echo "Region: ${REGION}"
echo "Cluster: ${CLUSTER_NAME}"
echo "Version: ${VERSION}"
echo ""

# Check prerequisites
command -v gcloud >/dev/null 2>&1 || { echo -e "${RED}Error: gcloud not installed${NC}" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}Error: kubectl not installed${NC}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Error: docker not installed${NC}" >&2; exit 1; }

# Set project
echo -e "${YELLOW}Setting GCP project...${NC}"
gcloud config set project ${PROJECT_ID}

# Get cluster credentials
echo -e "${YELLOW}Getting GKE credentials...${NC}"
gcloud container clusters get-credentials ${CLUSTER_NAME} --region ${REGION}

# Build and push images
REGISTRY="${REGION}-docker.pkg.dev/${PROJECT_ID}/weather-server"

echo -e "${YELLOW}Building Docker images...${NC}"
docker build -f Dockerfile.server -t ${REGISTRY}/weather-server:${VERSION} .
docker build -f Dockerfile.aggregator -t ${REGISTRY}/weather-aggregator:${VERSION} .
docker build -f Dockerfile.alarming -t ${REGISTRY}/weather-alarming:${VERSION} .
docker build -f Dockerfile.notification -t ${REGISTRY}/weather-notification:${VERSION} .

echo -e "${YELLOW}Pushing images to Artifact Registry...${NC}"
docker push ${REGISTRY}/weather-server:${VERSION}
docker push ${REGISTRY}/weather-aggregator:${VERSION}
docker push ${REGISTRY}/weather-alarming:${VERSION}
docker push ${REGISTRY}/weather-notification:${VERSION}

# Update deployments
echo -e "${YELLOW}Updating Kubernetes deployments...${NC}"
kubectl set image deployment/weather-server \
  server=${REGISTRY}/weather-server:${VERSION} \
  -n weather-system

kubectl set image deployment/weather-aggregator \
  aggregator=${REGISTRY}/weather-aggregator:${VERSION} \
  -n weather-system

kubectl set image deployment/weather-alarming \
  alarming=${REGISTRY}/weather-alarming:${VERSION} \
  -n weather-system

kubectl set image deployment/weather-notification \
  notification=${REGISTRY}/weather-notification:${VERSION} \
  -n weather-system

# Wait for rollout
echo -e "${YELLOW}Waiting for rollout to complete...${NC}"
kubectl rollout status deployment/weather-server -n weather-system
kubectl rollout status deployment/weather-aggregator -n weather-system
kubectl rollout status deployment/weather-alarming -n weather-system
kubectl rollout status deployment/weather-notification -n weather-system

# Get service endpoint
EXTERNAL_IP=$(kubectl get svc weather-server-service -n weather-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo ""
echo -e "${GREEN}✓ Deployment complete!${NC}"
echo "TCP Server endpoint: ${EXTERNAL_IP}:8080"
echo ""
echo "Check status:"
echo "  kubectl get pods -n weather-system"
echo "  kubectl logs -f deployment/weather-server -n weather-system"

