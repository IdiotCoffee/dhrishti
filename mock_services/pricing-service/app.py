import random

from flask import Flask, jsonify

from common.latency import simulate

app = Flask(__name__)

FLASH_PRICES = {"sku-1001": 29.99, "sku-1004": 149.99, "sku-1005": 19.99}
BASE_PRICES = {"sku-1002": 89.99, "sku-1003": 74.99}


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "pricing-service"})


@app.route("/price/<product_id>")
def price(product_id):
    simulate("fast", spike_chance=0.05)
    if product_id in FLASH_PRICES:
        return jsonify({
            "product_id": product_id,
            "price": FLASH_PRICES[product_id],
            "flash_sale": True,
            "discount_pct": random.randint(40, 70),
        })
    return jsonify({
        "product_id": product_id,
        "price": BASE_PRICES.get(product_id, round(random.uniform(19.99, 199.99), 2)),
        "flash_sale": False,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
