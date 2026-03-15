"""
flood_producer.py
-----------------
Floods metrics.raw with messages as fast as possible to build up
consumer lag beyond the 5000 threshold — triggering KEDA to scale up.

Sends messages in large batches with linger_ms to maximise throughput.

Usage:
    python flood_producer.py [bootstrap] [count]
    python flood_producer.py 192.168.10.91:30092 20000
"""

import sys, json, time, random
from kafka import KafkaProducer

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
COUNT     = int(sys.argv[2]) if len(sys.argv) > 2 else 20000
TOPIC     = "metrics.raw"

METRICS  = ["cpu_usage", "memory_used", "http_requests", "p99_latency", "error_rate"]
SERVICES = ["order-svc", "inventory-svc", "payment-svc"]

# Tuned for high throughput:
#   linger_ms=50   → batch messages for 50ms before sending
#   batch_size=65536 → large batch buffer
#   compression=gzip → smaller network payload
producer = KafkaProducer(
    bootstrap_servers=BOOTSTRAP,
    value_serializer=lambda v: json.dumps(v).encode(),
    key_serializer=lambda k: k.encode(),
    acks=1,
    linger_ms=50,
    batch_size=65536,
    compression_type="gzip",
)

print(f"Flooding {COUNT} messages to '{TOPIC}' on {BOOTSTRAP}")
print("This will build lag > 5000 to trigger KEDA autoscale...\n")

start = time.time()
sent  = 0

for i in range(COUNT):
    metric  = random.choice(METRICS)
    payload = {
        "metric":    metric,
        "service":   random.choice(SERVICES),
        "value":     round(random.uniform(1, 100), 2),
        "timestamp": int(time.time() * 1000),
        "seq":       i,
    }
    producer.send(TOPIC, key=metric, value=payload)
    sent += 1

    # Progress every 1000 messages
    if sent % 1000 == 0:
        elapsed = time.time() - start
        rate    = sent / elapsed
        print(f"  sent {sent:>6} / {COUNT}  "
              f"({rate:.0f} msg/s)  "
              f"elapsed={elapsed:.1f}s")

producer.flush()
producer.close()

elapsed = time.time() - start
print(f"\n✓ Sent {sent} messages in {elapsed:.1f}s "
      f"({sent/elapsed:.0f} msg/s avg)")
print("\nNow watch KEDA scale up:")
print("  watch -n2 microk8s kubectl get pods -n kafka")
print("  python watch_scale.py 192.168.10.91:30092")
