#!/bin/bash
# Initialize Kafka topics for Weather Server
# This script creates the required topics with proper configuration

set -e

BOOTSTRAP_SERVER="kafka:9093"
TOPICS=(
  "weather.metrics.raw:10:1"  # topic:partitions:replication-factor
  "weather.alarms:10:1"
)

echo "Waiting for Kafka to be ready..."
# Wait for Kafka to be available (max 60 seconds)
for i in {1..60}; do
  if /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server $BOOTSTRAP_SERVER &>/dev/null; then
    echo "Kafka is ready!"
    break
  fi
  if [ $i -eq 60 ]; then
    echo "ERROR: Kafka failed to start within 60 seconds"
    exit 1
  fi
  echo "Waiting for Kafka... ($i/60)"
  sleep 1
done

echo ""
echo "Creating Kafka topics..."

for topic_config in "${TOPICS[@]}"; do
  IFS=':' read -r topic partitions replication <<< "$topic_config"
  
  # Check if topic already exists
  if /opt/kafka/bin/kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVER --list | grep -q "^${topic}$"; then
    echo "✓ Topic '$topic' already exists"
  else
    echo "Creating topic: $topic (partitions=$partitions, replication=$replication)"
    /opt/kafka/bin/kafka-topics.sh \
      --bootstrap-server $BOOTSTRAP_SERVER \
      --create \
      --topic "$topic" \
      --partitions "$partitions" \
      --replication-factor "$replication" \
      --config retention.ms=604800000 \
      --config compression.type=lz4
    echo "✓ Topic '$topic' created successfully"
  fi
done

echo ""
echo "=== Current Kafka Topics ==="
/opt/kafka/bin/kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVER --list

echo ""
echo "=== Topic Details ==="
for topic_config in "${TOPICS[@]}"; do
  IFS=':' read -r topic _ _ <<< "$topic_config"
  /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server $BOOTSTRAP_SERVER \
    --describe \
    --topic "$topic"
done

echo ""
echo "✓ Kafka topics initialization complete!"

