"""
02_producer.py
--------------
Sends 30 metric samples to 'metrics.raw'.
Key = metric name  →  same metric always lands on the same partition.

Usage:
    python 02_producer.py [bootstrap]
"""

import sys, json, time, random
from kafka import KafkaProducer

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
TOPIC     = "metrics.raw"

METRICS  = ["cpu_usage", "memory_used", "http_requests", "p99_latency", "error_rate"]
SERVICES = ["order-svc", "inventory-svc", "payment-svc"]

producer = KafkaProducer(
    bootstrap_servers=BOOTSTRAP,
    value_serializer=lambda v: json.dumps(v).encode(),
    key_serializer=lambda k: k.encode(),
    acks=1,
)

print(f"Sending 30 messages → {TOPIC}  ({BOOTSTRAP})\n")
print(f"{'KEY':<20} {'PARTITION':>9} {'OFFSET':>7}  VALUE")
print("─" * 65)

for i in range(30):
    metric  = random.choice(METRICS)
    service = random.choice(SERVICES)
    payload = {
        "metric":    metric,
        "service":   service,
        "value":     round(random.uniform(1, 100), 2),
        "timestamp": int(time.time() * 1000),
    }
    record = producer.send(TOPIC, key=metric, value=payload).get(timeout=10)
    print(f"{metric:<20} {record.partition:>9} {record.offset:>7}  "
          f"service={service} value={payload['value']}")
    time.sleep(0.15)

producer.flush()
producer.close()
print("\n✓ Done. Run 03_consumer.py to read them back.")
