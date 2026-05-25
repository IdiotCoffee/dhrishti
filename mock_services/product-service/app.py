import random
import time

import requests
from flask import Flask

app = Flask(__name__)

"""
Product service.

Depends on inventory service.

Creates chained service dependencies:

gateway -> product -> inventory
"""


@app.route("/product")
def product():

    time.sleep(random.uniform(0.1, 0.3))

    inventory = requests.get(
        "http://inventory-service:8080/inventory",
        headers={"Connection": "close"},
        timeout=2,
    )

    return {
        "product": "laptop",
        "inventory": inventory.json(),
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
