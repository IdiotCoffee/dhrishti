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
    time.sleep(random.uniform(1.0, 2.0))

    return {
        "authenticated": True,
        "user": "demo-user",
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
