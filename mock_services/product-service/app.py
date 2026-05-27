import random
import time

import requests
from flask import Flask, jsonify

app = Flask(__name__)


@app.route("/product")
def product():
    time.sleep(random.uniform(0.3, 0.6))

    try:
        inventory = requests.get(
            "http://inventory-service:8080/inventory",
            timeout=5,
        )
        inv_body = inventory.json()
    except requests.exceptions.RequestException as e:
        return jsonify({
            "product": "laptop",
            "inventory": {"error": str(e)},
        }), 502

    if inventory.status_code != 200:
        return jsonify({
            "product": "laptop",
            "inventory": inv_body,
        }), inventory.status_code

    return jsonify({
        "product": "laptop",
        "inventory": inv_body,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
