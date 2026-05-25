import random
import time

from flask import Flask

app = Flask(__name__)

"""
Payment service.

Simulates:
- external dependency,
- unstable latency,
- transient failures.

Excellent observability target.
"""


@app.route("/pay")
def pay():

    # simulate slow external payment provider
    time.sleep(random.uniform(0.2, 1.5))

    # intermittent payment failure
    if random.random() < 0.15:
        return {"status": "payment-failed"}, 500

    return {"status": "success"}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
