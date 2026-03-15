#!/bin/bash
set -e

echo "==> Deploying Kafka on microk8s..."
microk8s kubectl apply -f kafka-kraft.yaml

echo ""
echo "==> Waiting for Kafka pod to be ready (up to 120s)..."
microk8s kubectl wait --namespace kafka \
  --for=condition=ready pod \
  --selector=app=kafka \
  --timeout=120s

echo ""
echo "==> Kafka pod status:"
microk8s kubectl get pods -n kafka

echo ""
echo "==> Services (note NodePort 30092):"
microk8s kubectl get svc -n kafka

echo ""
echo "==> Getting microk8s node IP..."
NODE_IP=$(microk8s kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
echo "    Node IP: $NODE_IP"
echo "    Kafka bootstrap: $NODE_IP:30092"

echo ""
echo "==> Creating a test topic 'metrics.raw' with 3 partitions..."
microk8s kubectl exec -n kafka kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create \
    --topic metrics.raw \
    --partitions 3 \
    --replication-factor 1 \
    --if-not-exists

echo ""
echo "==> Creating topic 'order.events'..."
microk8s kubectl exec -n kafka kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create \
    --topic order.events \
    --partitions 3 \
    --replication-factor 1 \
    --if-not-exists

echo ""
echo "==> Listing all topics:"
microk8s kubectl exec -n kafka kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --list

echo ""
echo "✓ Kafka is ready."
echo "  Bootstrap server for Python clients: $NODE_IP:30092"