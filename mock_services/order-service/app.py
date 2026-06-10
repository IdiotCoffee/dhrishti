import time

from flask import Flask, jsonify, request

from common.http_client import get_json, post_json
from common.latency import simulate

app = Flask(__name__)

INVENTORY = "http://inventory-service:8080"
PAYMENT = "http://payment-service:8080"
SHIPPING = "http://shipping-service:8080"
NOTIFICATION = "http://notification-service:8080"


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "order-service"})


@app.route("/order")
@app.route("/orders", methods=["POST"])
def orders():
    simulate("slow", spike_chance=0.08)
    body = request.json or {}
    product_id = body.get("product_id", "sku-1001")

    inventory = post_json(
        f"{INVENTORY}/reserve/{product_id}",
        {"product_id": product_id},
        timeout=8,
        name="inventory",
    )

    payment = None
    for attempt in range(2):
        payment = get_json(f"{PAYMENT}/pay", timeout=8, name="payment")
        if payment.get("status") == 200:
            break
        time.sleep(0.15)

    shipping = get_json(f"{SHIPPING}/quote", timeout=6, name="shipping")
    post_json(
        f"{NOTIFICATION}/notify",
        {"type": "order_placed", "product_id": product_id},
        timeout=4,
        name="notification",
    )

    return jsonify({
        "order_id": "ord-92831",
        "inventory": inventory,
        "payment": payment,
        "shipping": shipping,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
