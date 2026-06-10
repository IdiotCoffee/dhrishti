from flask import Flask, jsonify, request

from common.latency import simulate

app = Flask(__name__)
_events = 0


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "analytics-service"})


@app.route("/events", methods=["POST"])
def events():
    global _events
    simulate("fast")
    _events += 1
    body = request.json or {}
    return jsonify({"recorded": True, "event": body.get("event", "unknown"), "total": _events})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
