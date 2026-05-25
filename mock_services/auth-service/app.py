import random
import time

from flask import Flask

app = Flask(__name__)

"""
Auth service.

Simulates:
- token validation,
- session checks,
- lightweight CPU work.
"""


@app.route("/auth")
def auth():

    # realistic small latency
    time.sleep(random.uniform(0.05, 0.2))

    return {
        "authenticated": True,
        "user": "demo-user",
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
