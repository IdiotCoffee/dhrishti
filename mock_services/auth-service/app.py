import random
import time

from flask import Flask

app = Flask(__name__)

"""
Auth service — holds the TCP connection open while "validating"
so the observability graph can show active_connections on the edge.
"""


@app.route("/auth")
def auth():
    # Keep this edge mostly fast (<1s p95 target), with rare slow spikes.
    if random.random() < 0.97:
        time.sleep(random.uniform(0.12, 0.45))
    else:
        time.sleep(random.uniform(1.2, 1.8))

    return {
        "authenticated": True,
        "user": "demo-user",
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
