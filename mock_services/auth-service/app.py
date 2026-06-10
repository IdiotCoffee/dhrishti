from flask import Flask, jsonify

from common.latency import simulate

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "auth-service"})


@app.route("/auth")
@app.route("/validate")
def validate():
    simulate("fast", spike_chance=0.03)
    return jsonify({"valid": True, "token": "mock-jwt", "user_id": "user-42"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
