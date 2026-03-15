"""
05_check_lag.py
---------------
Shows current consumer lag for every group across all topics.
Run this after 03 and 04 to see committed offsets vs end offsets.

Usage:
    python 05_check_lag.py [bootstrap]
"""

import sys
from kafka import KafkaAdminClient, KafkaConsumer, TopicPartition

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"
TOPICS    = ["metrics.raw", "order.events", "audit.log"]

admin  = KafkaAdminClient(bootstrap_servers=BOOTSTRAP)
groups = [g for g, _ in admin.list_consumer_groups()
          if not g.startswith("__")]

if not groups:
    print("No consumer groups found yet. Run 03_consumer.py first.")
    admin.close()
    sys.exit(0)

print(f"\n{'GROUP':<22} {'TOPIC':<15} {'P':>2} "
      f"{'END':>10} {'COMMITTED':>10} {'LAG':>8}")
print("─" * 72)

for group_id in sorted(groups):
    c = KafkaConsumer(bootstrap_servers=BOOTSTRAP, group_id=group_id)
    for topic in TOPICS:
        parts = c.partitions_for_topic(topic)
        if not parts:
            continue
        tps = [TopicPartition(topic, p) for p in sorted(parts)]
        ends = c.end_offsets(tps)
        for tp in tps:
            committed = c.committed(tp) or 0
            end       = ends[tp]
            lag       = max(0, end - committed)
            flag      = "  ⚠ lagging" if lag > 50 else ""
            print(f"{group_id:<22} {topic:<15} {tp.partition:>2} "
                  f"{end:>10} {committed:>10} {lag:>8}{flag}")
    c.close()

admin.close()
