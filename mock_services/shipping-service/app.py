import random

from flask import Flask, jsonify

from common.latency import simulate

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "shipping-service"})


@app.route("/quote")
def quote():
    simulate("normal", spike_chance=0.06)
    return jsonify({
        "carrier": random.choice(["fedex", "ups", "dhl"]),
        "days": random.randint(2, 7),
        "cost": round(random.uniform(4.99, 14.99), 2),
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
