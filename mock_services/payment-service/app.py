import random
import time

from flask import Flask

app = Flask(__name__)

"""
Payment — simulates a slow external provider; connection stays open for the sleep.
"""


@app.route("/pay")
def pay():
    # Keep payment mixed: often fast, sometimes slow enough to push p95 >1s.
    if random.random() < 0.8:
        time.sleep(random.uniform(0.15, 0.6))
    else:
        time.sleep(random.uniform(1.3, 2.4))

    if random.random() < 0.1:
        return {"status": "payment-failed"}, 500

    return {"status": "success"}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
