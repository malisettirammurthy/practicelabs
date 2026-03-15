"""
04_consumer_groups.py
---------------------
Runs TWO consumer groups in parallel threads reading the same topic.
Shows that each group receives every message independently.

  rollup-worker   → aggregates (simulates 1-min rollup)
  alert-evaluator → checks thresholds (fires alert if value > 80)

Usage:
    python 04_consumer_groups.py [bootstrap]

Press Ctrl-C to stop.
"""

import sys, json, time, threading, signal
from kafka import KafkaConsumer

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
TOPIC     = "metrics.raw"
running   = True

def stop(s, f): global running; running = False
signal.signal(signal.SIGINT, stop)
signal.signal(signal.SIGTERM, stop)


def run_group(group_id: str, label: str, handler):
    consumer = KafkaConsumer(
        TOPIC,
        bootstrap_servers=BOOTSTRAP,
        group_id=group_id,
        auto_offset_reset="earliest",
        enable_auto_commit=True,
        value_deserializer=lambda b: json.loads(b.decode()),
        key_deserializer=lambda b: b.decode() if b else None,
    )
    print(f"[{label}] started  group={group_id}")
    while running:
        for tp, msgs in consumer.poll(timeout_ms=500).items():
            for m in msgs:
                handler(label, m)
    consumer.close()
    print(f"[{label}] stopped")


# ── Rollup handler: count occurrences per metric ──
rollup_counts = {}
def rollup_handler(label, m):
    metric = m.value.get("metric", "?")
    rollup_counts[metric] = rollup_counts.get(metric, 0) + 1
    print(f"  [{label}|P{m.partition}] {metric:<20} "
          f"value={m.value.get('value'):<8} "
          f"seen={rollup_counts[metric]}x")


# ── Alert handler: fire alert if value > 80 ──
def alert_handler(label, m):
    val  = m.value.get("value", 0)
    flag = "  ⚠  ALERT" if val > 80 else ""
    print(f"  [{label}|P{m.partition}] {m.value.get('metric'):<20} "
          f"value={val:<8}{flag}")


t1 = threading.Thread(target=run_group,
                      args=("rollup-worker",   "ROLLUP", rollup_handler),
                      daemon=True)
t2 = threading.Thread(target=run_group,
                      args=("alert-evaluator", "ALERT",  alert_handler),
                      daemon=True)

t1.start(); t2.start()
print(f"Two groups reading '{TOPIC}'.  Ctrl-C to stop.\n")

while running:
    time.sleep(0.5)

t1.join(timeout=3)
t2.join(timeout=3)
print("\nFinal rollup counts:", rollup_counts)
