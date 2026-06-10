from flask import Flask, jsonify

from common.latency import simulate

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "user-service"})


@app.route("/profile")
def profile():
    simulate("db", spike_chance=0.05)
    return jsonify({
        "user_id": "user-42",
        "name": "Alex Rivera",
        "tier": "gold",
        "addresses": 2,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
