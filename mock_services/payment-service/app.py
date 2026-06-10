from flask import Flask, jsonify

from common.latency import maybe_fail, simulate

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "payment-service"})


@app.route("/pay")
def pay():
    simulate("normal", spike_chance=0.15)
    if maybe_fail(0.09):
        return jsonify({"error": "payment gateway timeout"}), 500
    return jsonify({"charged": True, "amount": 29.99, "provider": "mock-stripe"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
