import os
import time

from flask import Flask, jsonify, request

from common.fanout import parallel
from common.http_client import get_json, post_json
from common.latency import simulate

app = Flask(__name__)

INVENTORY = os.getenv("INVENTORY_SERVICE_URL", "http://inventory-service:8080")
PRICING = os.getenv("PRICING_SERVICE_URL", "http://pricing-service:8080")

SALE_START = int(os.getenv("FLASH_SALE_START_EPOCH", "0"))
SALE_DURATION = int(os.getenv("FLASH_SALE_DURATION_SEC", "3600"))

FLASH_SKUS = ["sku-1001", "sku-1004", "sku-1005"]


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "flash-sale-service"})


def sale_active():
    if SALE_START <= 0:
        return True
    now = int(time.time())
    return SALE_START <= now < SALE_START + SALE_DURATION


def _sku_snapshot(sku):
    return {
        "sku": sku,
        "stock": get_json(f"{INVENTORY}/stock/{sku}", timeout=6, name="inventory"),
        "price": get_json(f"{PRICING}/price/{sku}", timeout=5, name="pricing"),
    }


@app.route("/active")
def active():
    simulate("flash", spike_chance=0.12)
    items = parallel({sku: (lambda s=sku: _sku_snapshot(s)) for sku in FLASH_SKUS})
    return jsonify({
        "active": sale_active(),
        "items": [items[sku] for sku in FLASH_SKUS],
        "ends_in_sec": max(0, SALE_START + SALE_DURATION - int(time.time())) if SALE_START > 0 else 3600,
    })


@app.route("/reserve/<product_id>", methods=["POST"])
def reserve(product_id):
    simulate("flash", spike_chance=0.2)
    if not sale_active():
        return jsonify({"error": "sale not active"}), 400
    body = request.get_json(silent=True) or {}
    results = parallel({
        "inventory": lambda: post_json(
            f"{INVENTORY}/reserve/{product_id}",
            body,
            timeout=8,
            name="inventory",
        ),
        "pricing": lambda: get_json(f"{PRICING}/price/{product_id}", timeout=5, name="pricing"),
    })
    return jsonify({"reservation": results["inventory"], "price": results["pricing"]})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
