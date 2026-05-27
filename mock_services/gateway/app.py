import random

import requests
from flask import Flask, jsonify

app = Flask(__name__)

"""
Gateway fans out sequentially — client→gateway stays open for the full fanout.
"""


def fetch_json(name, url, timeout):
    try:
        resp = requests.get(url, timeout=timeout)
        try:
            body = resp.json()
        except ValueError:
            body = {"raw": resp.text[:200]}

        return {"status": resp.status_code, "body": body}
    except requests.exceptions.Timeout:
        return {"status": 504, "error": f"{name} timed out"}
    except requests.exceptions.RequestException as e:
        return {"status": 502, "error": f"{name} unreachable: {e}"}


@app.route("/")
def home():
    responses = {}

    responses["auth"] = fetch_json(
        "auth-service",
        "http://auth-service:8080/auth",
        timeout=5,
    )

    responses["product"] = fetch_json(
        "product-service",
        "http://product-service:8080/product",
        timeout=8,
    )

    if random.random() < 0.7:
        responses["order"] = fetch_json(
            "order-service",
            "http://order-service:8080/order",
            timeout=15,
        )

    return jsonify(responses)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
