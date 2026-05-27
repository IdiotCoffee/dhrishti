import random
import time

from flask import Flask

app = Flask(__name__)

"""
Inventory — slow responses + occasional failures for observability demos.
"""


@app.route("/inventory")
def inventory():
    time.sleep(random.uniform(1.0, 2.5))

    if random.random() < 0.08:
        return {"error": "inventory timeout"}, 500

    return {
        "stock": random.randint(1, 50),
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
