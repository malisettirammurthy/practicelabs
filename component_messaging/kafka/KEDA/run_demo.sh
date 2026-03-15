#!/bin/bash
# run_demo.sh
# Full walkthrough of the KEDA autoscale demo.
# Run each step in order — the script pauses and tells you what to do.
set -e

NODE_IP=$(microk8s kubectl get nodes \
  -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
BOOTSTRAP="$NODE_IP:30092"

echo "════════════════════════════════════════════"
echo "  KEDA Kafka Autoscale Demo"
echo "  Bootstrap: $BOOTSTRAP"
echo "════════════════════════════════════════════"
echo ""

# ── Step 1: Deploy consumer + ScaledObject ──
echo "STEP 1 — Deploying consumer Deployment and KEDA ScaledObject..."
microk8s kubectl apply -f keda-demo.yaml
echo ""

echo "Waiting for consumer pod to be ready..."
microk8s kubectl wait --namespace kafka \
  --for=condition=ready pod \
  --selector=app=rollup-consumer \
  --timeout=120s
echo ""

echo "Current state (1 replica, no lag):"
microk8s kubectl get pods -n kafka -l app=rollup-consumer
echo ""
microk8s kubectl get scaledobject -n kafka
echo ""

# ── Step 2: Show initial lag = 0 ──
echo "STEP 2 — Checking initial lag (should be 0)..."
python3 watch_scale.py $BOOTSTRAP &
WATCH_PID=$!
sleep 6
echo ""

# ── Step 3: Flood the topic ──
echo "STEP 3 — Flooding topic with 20,000 messages to build lag..."
echo "         Watch the replica count go from 1 → 2 → 3"
echo ""
python3 flood_producer.py $BOOTSTRAP 20000
echo ""

echo "Watching for 60 seconds — KEDA should scale up then back down..."
sleep 60

# ── Step 4: Show scale-down ──
echo ""
echo "STEP 4 — Consumers drained the lag. Watching scale-down (cooldown=30s)..."
sleep 40

kill $WATCH_PID 2>/dev/null || true

echo ""
echo "════════════════════════════════════════════"
echo "  Demo complete."
echo "  Final pod count:"
microk8s kubectl get pods -n kafka -l app=rollup-consumer
echo ""
echo "  ScaledObject status:"
microk8s kubectl get scaledobject -n kafka
echo "════════════════════════════════════════════"
