"""
03_consumer.py
--------------
Reads from 'metrics.raw'.
Prints which partition each message came from and commits manually.

Usage:
    python 03_consumer.py [bootstrap] [group_id]

Press Ctrl-C to stop.
"""

import sys, json, signal
from kafka import KafkaConsumer
from kafka.structs import OffsetAndMetadata


BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
GROUP_ID  = sys.argv[2] if len(sys.argv) > 2 else "rollup-worker"
TOPIC     = "metrics.raw"

running = True
def stop(s, f): global running; running = False
signal.signal(signal.SIGINT, stop)
signal.signal(signal.SIGTERM, stop)

consumer = KafkaConsumer(
    TOPIC,
    bootstrap_servers=BOOTSTRAP,
    group_id=GROUP_ID,
    auto_offset_reset="earliest",      # read from start on first run
    enable_auto_commit=False,          # we commit manually
    value_deserializer=lambda b: json.loads(b.decode()),
    key_deserializer=lambda b: b.decode() if b else None,
)

print(f"group={GROUP_ID}  topic={TOPIC}  bootstrap={BOOTSTRAP}")
print("Waiting for messages ... (Ctrl-C to stop)\n")
print(f"{'P':>2} {'OFFSET':>7}  {'KEY':<20} {'SERVICE':<15} VALUE")
print("─" * 65)

count = 0
while running:
    batch = consumer.poll(timeout_ms=1000, max_records=20)
    for tp, msgs in batch.items():
        for m in msgs:
            count += 1
            v = m.value
            print(f"{m.partition:>2} {m.offset:>7}  "
                  f"{m.key:<20} {v.get('service',''):<15} "
                  f"{v.get('metric')}={v.get('value')}")
        # manual commit after processing each partition batch
        # consumer.commit({tp: msgs[-1].offset + 1})
        consumer.commit({
            tp: OffsetAndMetadata(msgs[-1].offset + 1, None)
            })

consumer.close()
print(f"\n✓ Received {count} messages total.")
