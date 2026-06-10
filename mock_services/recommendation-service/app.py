import random

from flask import Flask, jsonify

from common.http_client import get_json
from common.latency import simulate

app = Flask(__name__)
CATALOG = "http://product-catalog:8080"


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "recommendation-service"})


@app.route("/recommendations")
def recommendations():
    simulate("db", spike_chance=0.1)
    catalog = get_json(f"{CATALOG}/featured", timeout=5, name="product-catalog")
    return jsonify({
        "strategy": random.choice(["collaborative", "trending", "flash-picks"]),
        "items": catalog,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
