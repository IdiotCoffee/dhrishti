import random
import time

from flask import Flask

app = Flask(__name__)

"""
Payment — simulates a slow external provider; connection stays open for the sleep.
"""


@app.route("/pay")
def pay():
    time.sleep(random.uniform(1.0, 2.0))

    if random.random() < 0.1:
        return {"status": "payment-failed"}, 500

    return {"status": "success"}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
