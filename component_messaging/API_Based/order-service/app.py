from flask import Flask, request, jsonify
import requests
import os
import uuid
from datetime import datetime

app = Flask(__name__)

orders_db = []

NOTIFICATION_SERVICE_URL = os.getenv("NOTIFICATION_SERVICE_URL", "http://localhost:8081")
ANALYTICS_SERVICE_URL = os.getenv("ANALYTICS_SERVICE_URL", "http://localhost:8082")


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
            return jsonify({
                "error": "customer_id and product_id are required"
            }), 400

        order = {
            "order_id": str(uuid.uuid4()),
            "customer_id": customer_id,
            "product_id": product_id,
            "quantity": quantity,
            "status": "CREATED",
            "created_at": datetime.now().isoformat() + "Z"
        }

        orders_db.append(order)

        notification_payload = {
            "event_type": "ORDER_CREATED",
            "order_id": order["order_id"],
            "customer_id": order["customer_id"],
            "message": f"Order {order['order_id']} created successfully"
        }

        analytics_payload = {
            "event_type": "ORDER_CREATED",
            "order_id": order["order_id"],
            "customer_id": order["customer_id"],
            "product_id": order["product_id"],
            "quantity": order["quantity"],
            "timestamp": order["created_at"]
        }

        notification_result = {"status": "not_called"}
        analytics_result = {"status": "not_called"}

        try:
            resp = requests.post(
                f"{NOTIFICATION_SERVICE_URL}/notify",
                json=notification_payload,
                timeout=3
            )
            notification_result = {
                "status_code": resp.status_code,
                "response": resp.json() if resp.content else {}
            }
        except Exception as exc:
            notification_result = {"error": str(exc)}

        try:
            resp = requests.post(
                f"{ANALYTICS_SERVICE_URL}/events",
                json=analytics_payload,
                timeout=3
            )
            analytics_result = {
                "status_code": resp.status_code,
                "response": resp.json() if resp.content else {}
            }
        except Exception as exc:
            analytics_result = {"error": str(exc)}

        return jsonify({
            "message": "Order created",
            "order": order,
            "downstream_calls": {
                "notification_service": notification_result,
                "analytics_service": analytics_result
            }
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