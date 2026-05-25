import random
import time

from flask import Flask

app = Flask(__name__)

"""
Inventory service.

This intentionally introduces:
- random latency,
- intermittent failures.

Why?

Because observability becomes interesting only when
systems behave unpredictably.
"""


@app.route("/inventory")
def inventory():

    # random latency spike
    latency = random.uniform(0.05, 0.8)
    time.sleep(latency)

    # intermittent failure
    if random.random() < 0.1:
        return {"error": "inventory timeout"}, 500

    return {
        "stock": random.randint(1, 50),
        "latency": latency,
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
