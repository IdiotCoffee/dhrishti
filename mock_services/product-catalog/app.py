import random

from flask import Flask, jsonify

from common.latency import simulate

app = Flask(__name__)

PRODUCTS = [
    {"id": "sku-1001", "name": "Wireless Earbuds Pro", "category": "electronics"},
    {"id": "sku-1002", "name": "Mechanical Keyboard", "category": "electronics"},
    {"id": "sku-1003", "name": "Running Shoes X", "category": "apparel"},
    {"id": "sku-1004", "name": "Smart Watch S9", "category": "electronics"},
    {"id": "sku-1005", "name": "Flash Deal Backpack", "category": "accessories"},
]


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "product-catalog"})


@app.route("/product")
@app.route("/products")
def products():
    simulate("normal", spike_chance=0.12)
    return jsonify({"items": PRODUCTS, "count": len(PRODUCTS)})


@app.route("/products/<product_id>")
def product(product_id):
    simulate("normal", spike_chance=0.1)
    item = next((p for p in PRODUCTS if p["id"] == product_id), None)
    if not item:
        return jsonify({"error": "not found"}), 404
    return jsonify(item)


@app.route("/featured")
def featured():
    simulate("fast", spike_chance=0.08)
    return jsonify({"featured": random.sample(PRODUCTS, k=3)})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
