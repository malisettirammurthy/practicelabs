"""
01_create_topics.py
-------------------
Creates all topics needed for the samples.
Uses the Python admin client — no kubectl exec required at all.

Usage:
    python 01_create_topics.py [bootstrap]
    python 01_create_topics.py 192.168.10.91:30092
"""

import sys
from kafka.admin import KafkaAdminClient, NewTopic
from kafka.errors import TopicAlreadyExistsError

BOOTSTRAP = sys.argv[1] if len(sys.argv) > 1 else "localhost:30092"

TOPICS = [
    NewTopic(name="metrics.raw",  num_partitions=3, replication_factor=1),
    NewTopic(name="order.events", num_partitions=3, replication_factor=1),
    NewTopic(name="audit.log",    num_partitions=3, replication_factor=1),
]

print(f"Connecting to Kafka at {BOOTSTRAP} ...")
admin = KafkaAdminClient(
    bootstrap_servers=BOOTSTRAP,
    client_id="admin-setup",
)

print("Creating topics ...")
for t in TOPICS:
    try:
        admin.create_topics([t])
        print(f"  ✓ created  {t.name}  "
              f"(partitions={t.num_partitions}, "
              f"replication={t.replication_factor})")
    except TopicAlreadyExistsError:
        print(f"  ~ exists   {t.name}")
    except Exception as e:
        print(f"  ✗ error    {t.name}: {e}")

print("\nAll topics on broker:")
all_topics = sorted(
    t for t in admin.list_topics()
    if not t.startswith("__")        # hide internal topics
)
for t in all_topics:
    print(f"  {t}")

admin.close()
print("\nDone. Run 02_producer.py next.")