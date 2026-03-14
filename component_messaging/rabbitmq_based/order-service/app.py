from flask import Flask, request, jsonify
import os
import uuid
from datetime import datetime
import pika
import json

app = Flask(__name__)

orders_db = []

RABBITMQ_HOST = os.getenv("RABBITMQ_HOST", "localhost")
RABBITMQ_PORT = int(os.getenv("RABBITMQ_PORT", "5672"))
RABBITMQ_USERNAME = os.getenv("RABBITMQ_USERNAME", "guest")
RABBITMQ_PASSWORD = os.getenv("RABBITMQ_PASSWORD", "guest")

EXCHANGE_NAME = "order.events"


def publish_order_event(order):
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

    notification_event = {
        "event_type": "ORDER_CREATED",
        "target": "notification",
        "order": order,
        "message": f"Order {order['order_id']} created successfully"
    }

    analytics_event = {
        "event_type": "ORDER_CREATED",
        "target": "analytics",
        "order": order
    }

    channel.basic_publish(
        exchange=EXCHANGE_NAME,
        routing_key="order.created.notification",
        body=json.dumps(notification_event),
        properties=pika.BasicProperties(
            delivery_mode=2
        )
    )

    channel.basic_publish(
        exchange=EXCHANGE_NAME,
        routing_key="order.created.analytics",
        body=json.dumps(analytics_event),
        properties=pika.BasicProperties(
            delivery_mode=2
        )
    )

    connection.close()


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"service": "order-service", "status": "up"}), 200


@app.route("/orders", methods=["POST"])
def create_order():
    try:
        data = request.get_json(force=True)

        customer_id = data.get("customer_id")
        product_id = data.get("product_id")
        quantity = data.get("quantity", 1)

        if not customer_id or not product_id:
            return jsonify({"error": "customer_id and product_id are required"}), 400

        order = {
            "order_id": str(uuid.uuid4()),
            "customer_id": customer_id,
            "product_id": product_id,
            "quantity": quantity,
            "status": "CREATED",
            "created_at": datetime.utcnow().isoformat() + "Z"
        }

        orders_db.append(order)

        publish_order_event(order)

        return jsonify({
            "message": "Order created and events published",
            "order": order
        }), 201

    except Exception as exc:
        return jsonify({"error": str(exc)}), 500


@app.route("/orders", methods=["GET"])
def list_orders():
    return jsonify({
        "count": len(orders_db),
        "orders": orders_db
    }), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, debug=True)