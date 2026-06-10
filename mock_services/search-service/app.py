from flask import Flask, jsonify, request

from common.http_client import get_json
from common.latency import simulate

app = Flask(__name__)
CATALOG = "http://product-catalog:8080"


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "search-service"})


@app.route("/search")
def search():
    simulate("db", spike_chance=0.12)
    query = request.args.get("q", "")
    catalog = get_json(f"{CATALOG}/products", timeout=6, name="product-catalog")
    return jsonify({"query": query, "results": catalog, "hits": 5})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
