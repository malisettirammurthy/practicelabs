from flask import Flask, request, jsonify
from datetime import datetime

app = Flask(__name__)

events_db = []


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"service": "analytics-service", "status": "up"}), 200


@app.route("/events", methods=["POST"])
def ingest_event():
    try:
        data = request.get_json(force=True)

        required_fields = ["event_type", "order_id", "customer_id"]
        missing = [field for field in required_fields if not data.get(field)]

        if missing:
            return jsonify({
                "error": f"missing required fields: {', '.join(missing)}"
            }), 400

        event = {
            "event_type": data["event_type"],
            "order_id": data["order_id"],
            "customer_id": data["customer_id"],
            "product_id": data.get("product_id"),
            "quantity": data.get("quantity"),
            "timestamp": data.get("timestamp"),
            "received_at": datetime.utcnow().isoformat() + "Z"
        }

        events_db.append(event)

        return jsonify({
            "message": "analytics event ingested",
            "event": event
        }), 202

    except Exception as exc:
        return jsonify({"error": str(exc)}), 500


@app.route("/events", methods=["GET"])
def list_events():
    return jsonify({
        "count": len(events_db),
        "events": events_db
    }), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8082, debug=True)