from flask import Flask, jsonify
from datetime import datetime
import threading
import pika
import json
import os
import time

app = Flask(__name__)

events_db = []

RABBITMQ_HOST = os.getenv("RABBITMQ_HOST", "localhost")
RABBITMQ_PORT = int(os.getenv("RABBITMQ_PORT", "5672"))
RABBITMQ_USERNAME = os.getenv("RABBITMQ_USERNAME", "guest")
RABBITMQ_PASSWORD = os.getenv("RABBITMQ_PASSWORD", "guest")

EXCHANGE_NAME = "order.events"
QUEUE_NAME = "analytics.queue"
ROUTING_KEY = "order.created.analytics"


def rabbitmq_consumer():
    while True:
        try:
            credentials = pika.PlainCredentials(RABBITMQ_USERNAME, RABBITMQ_PASSWORD)
            parameters = pika.ConnectionParameters(
                host=RABBITMQ_HOST,
                port=RABBITMQ_PORT,
                credentials=credentials
            )

            connection = pika.BlockingConnection(parameters)
            channel = connection.channel()

            channel.exchange_declare(
                exchange=EXCHANGE_NAME,
                exchange_type="direct",
                durable=True
            )

            channel.queue_declare(queue=QUEUE_NAME, durable=True)
            channel.queue_bind(
                exchange=EXCHANGE_NAME,
                queue=QUEUE_NAME,
                routing_key=ROUTING_KEY
            )

            def callback(ch, method, properties, body):
                try:
                    payload = json.loads(body.decode("utf-8"))
                    event = {
                        "event_type": payload.get("event_type"),
                        "order": payload.get("order"),
                        "received_at": datetime.utcnow().isoformat() + "Z"
                    }
                    events_db.append(event)
                    print(f"Analytics consumed: {event}")
                    ch.basic_ack(delivery_tag=method.delivery_tag)
                except Exception as exc:
                    print(f"Failed to process message: {exc}")
                    ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

            channel.basic_qos(prefetch_count=1)
            channel.basic_consume(queue=QUEUE_NAME, on_message_callback=callback)

            print("Waiting for analytics messages...")
            channel.start_consuming()

        except Exception as exc:
            print(f"RabbitMQ connection/consumer error: {exc}")
            time.sleep(5)


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"service": "analytics-service", "status": "up"}), 200


@app.route("/events", methods=["GET"])
def list_events():
    return jsonify({
        "count": len(events_db),
        "events": events_db
    }), 200


if __name__ == "__main__":
    thread = threading.Thread(target=rabbitmq_consumer, daemon=True)
    thread.start()
    app.run(host="0.0.0.0", port=8082, debug=True)