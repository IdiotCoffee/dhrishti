import random
import threading

from flask import Flask, jsonify

from common.latency import maybe_fail, simulate

app = Flask(__name__)

_lock = threading.Lock()
_flash_stock = {"sku-1001": 500, "sku-1004": 200, "sku-1005": 1000}
_regular_stock = {p: random.randint(20, 200) for p in ["sku-1002", "sku-1003"]}


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "inventory-service"})


@app.route("/inventory")
@app.route("/stock/<product_id>")
def stock(product_id=None):
    simulate("db", spike_chance=0.18)
    if maybe_fail(0.06):
        return jsonify({"error": "inventory shard timeout"}), 500

    with _lock:
        if product_id and product_id in _flash_stock:
            remaining = _flash_stock[product_id]
            return jsonify({"product_id": product_id, "stock": remaining, "flash": True})
        qty = _regular_stock.get(product_id, random.randint(5, 80))
        return jsonify({"product_id": product_id or "all", "stock": qty})


@app.route("/reserve/<product_id>", methods=["POST"])
def reserve(product_id):
    simulate("flash", spike_chance=0.25)
    with _lock:
        if product_id in _flash_stock:
            if _flash_stock[product_id] <= 0:
                return jsonify({"reserved": False, "reason": "sold_out"}), 409
            _flash_stock[product_id] -= 1
            return jsonify({"reserved": True, "remaining": _flash_stock[product_id]})
    return jsonify({"reserved": True, "remaining": random.randint(1, 50)})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
