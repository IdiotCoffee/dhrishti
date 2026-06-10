import random

from flask import Flask, jsonify, request

from common.http_client import get_json
from common.latency import simulate

app = Flask(__name__)
INVENTORY = "http://inventory-service:8080"


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "cart-service"})


@app.route("/items", methods=["GET", "POST"])
def items():
    simulate("normal", spike_chance=0.1)
    if request.method == "POST":
        body = request.json or {}
        product_id = body.get("product_id", "sku-1001")
        stock = get_json(f"{INVENTORY}/stock/{product_id}", timeout=5, name="inventory")
        return jsonify({
            "cart_id": f"cart-{random.randint(1000, 9999)}",
            "item": product_id,
            "quantity": body.get("quantity", 1),
            "inventory": stock,
        })
    return jsonify({"items": [{"product_id": "sku-1001", "qty": 1}], "total": 29.99})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
