import random

import requests
from flask import Flask

app = Flask(__name__)

"""
Gateway service.

Acts like:
- ingress layer,
- API aggregation service,
- request fanout node.

This creates multiple downstream dependencies
from a single incoming request.
"""


@app.route("/")
def home():

    responses = {}

    # auth validation
    auth = requests.get(
        "http://auth-service:8080/auth",
        headers={"Connection": "close"},
        timeout=2,
    )

    responses["auth"] = auth.json()

    # product lookup
    product = requests.get(
        "http://product-service:8080/product",
        headers={"Connection": "close"},
        timeout=2,
    )

    responses["product"] = product.json()

    # randomly simulate order creation
    if random.random() < 0.7:
        order = requests.get(
            "http://order-service:8080/order",
            headers={"Connection": "close"},
            timeout=3,
        )

        responses["order"] = order.json()

    return responses


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
