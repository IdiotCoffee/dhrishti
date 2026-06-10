from flask import Flask, jsonify, request

from common.latency import simulate

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "notification-service"})


@app.route("/notify", methods=["POST"])
def notify():
    simulate("fast", spike_chance=0.04)
    body = request.json or {}
    return jsonify({"sent": True, "channel": "email", "type": body.get("type", "generic")})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
