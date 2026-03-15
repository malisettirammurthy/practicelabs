"""
watch_scale.py
--------------
Polls every 3 seconds and prints:
  - current replica count of the Deployment
  - lag per partition for the consumer group
  - KEDA ScaledObject status

Shows the full autoscale lifecycle:
  1. lag builds up  → replicas increase
  2. consumers drain the lag
  3. lag drops      → replicas scale back down

Usage:
    python watch_scale.py [bootstrap]

Run this in a separate terminal while flood_producer.py is running.
"""

import sys, time, subprocess, json
from kafka import KafkaConsumer, TopicPartition
from kafka.admin import KafkaAdminClient

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
GROUP     = "rollup-worker"
TOPIC     = "metrics.raw"


def get_replicas():
    """Ask kubectl for current Deployment replica counts."""
    try:
        r = subprocess.run(
            ["microk8s", "kubectl", "get", "deployment",
             "rollup-consumer", "-n", "kafka",
             "-o", "jsonpath={.spec.replicas},{.status.readyReplicas}"],
            capture_output=True, text=True, timeout=5
        )
        parts = r.stdout.strip().split(",")
        desired = int(parts[0]) if parts[0] else 0
        ready   = int(parts[1]) if len(parts) > 1 and parts[1] else 0
        return desired, ready
    except Exception as e:
        return -1, -1


def get_scaledobject_status():
    """Get KEDA ScaledObject status."""
    try:
        r = subprocess.run(
            ["microk8s", "kubectl", "get", "scaledobject",
             "rollup-consumer-scaler", "-n", "kafka",
             "-o", "jsonpath={.status.conditions[0].type}={.status.conditions[0].status}"],
            capture_output=True, text=True, timeout=5
        )
        return r.stdout.strip() or "unknown"
    except:
        return "unknown"


def get_lag():
    """Calculate total lag across all partitions for the group."""
    try:
        c = KafkaConsumer(
            bootstrap_servers=BOOTSTRAP,
            group_id=GROUP,
            consumer_timeout_ms=3000,
        )
        parts = c.partitions_for_topic(TOPIC) or set()
        tps   = [TopicPartition(TOPIC, p) for p in sorted(parts)]
        ends  = c.end_offsets(tps)
        rows  = []
        total = 0
        for tp in tps:
            committed = c.committed(tp) or 0
            end       = ends[tp]
            lag       = max(0, end - committed)
            total    += lag
            rows.append((tp.partition, end, committed, lag))
        c.close()
        return total, rows
    except Exception as e:
        return -1, []


print(f"Watching KEDA autoscale for group='{GROUP}' on '{TOPIC}'")
print(f"Bootstrap: {BOOTSTRAP}")
print(f"Scale trigger: lag > 5000 per replica\n")
print(f"{'TIME':>8}  {'DESIRED':>7} {'READY':>5}  "
      f"{'TOTAL LAG':>10}  {'P0 lag':>8} {'P1 lag':>8} {'P2 lag':>8}  KEDA")
print("─" * 85)

iteration = 0
while True:
    iteration += 1
    ts               = time.strftime("%H:%M:%S")
    desired, ready   = get_replicas()
    total_lag, rows  = get_lag()
    keda_status      = get_scaledobject_status()

    p_lags = {r[0]: r[3] for r in rows}
    p0 = p_lags.get(0, 0)
    p1 = p_lags.get(1, 0)
    p2 = p_lags.get(2, 0)

    # Visual indicator
    if total_lag > 5000:
        indicator = "⬆ scaling up"
    elif total_lag > 0:
        indicator = "↓ draining"
    else:
        indicator = "✓ idle"

    print(f"{ts:>8}  {desired:>7} {ready:>5}  "
          f"{total_lag:>10,}  {p0:>8,} {p1:>8,} {p2:>8,}  "
          f"{indicator}  [{keda_status}]",
          flush=True)

    time.sleep(3)
