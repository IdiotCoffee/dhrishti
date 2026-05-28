import random
import time

from flask import Flask

app = Flask(__name__)

"""
Inventory — slow responses + occasional failures for observability demos.
"""


@app.route("/inventory")
def inventory():
    # Bimodal latency so this edge has visible tail behavior.
    if random.random() < 0.75:
        time.sleep(random.uniform(0.2, 0.75))
    else:
        time.sleep(random.uniform(1.6, 3.0))

    if random.random() < 0.08:
        return {"error": "inventory timeout"}, 500

    return {
        "stock": random.randint(1, 50),
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
