# Shared Kafka Infrastructure

This directory contains the shared Kafka broker used by multiple microservices.

## Setup

### 1. Format Kafka Storage (First Time Only)
bash

docker run --rm apache/kafka:4.1.1 /opt/kafka/bin/kafka-storage.sh random-uuid

docker run --rm \
  -v kafka-data:/var/lib/kafka/data \
  -e KAFKA_CLUSTER_ID="PKFpmUiNSOyhc_fn7Sy6Rg" \
  apache/kafka:4.1.1 \
  /opt/kafka/bin/kafka-storage.sh format --cluster-id PKFpmUiNSOyhc_fn7Sy6Rg --ignore-formatted --config /opt/kafka/config/server.properties --standalone




bash
cd ~/Desktop/shared-kafka
docker-compose up -d


docker exec shared-kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic jar-events \
  --partitions 1 \
  --replication-factor 1 \
  --bootstrap-server shared-kafka:9092


when self hosting to a prod server change this following variable 
# KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://<YOUR_SERVER_IP_OR_HOSTNAME>:9092,EXTERNAL://<YOUR_SERVER_IP_OR_HOSTNAME>:19092
