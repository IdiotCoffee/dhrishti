import os
import random
import threading

from flask import Flask, jsonify, request

from common.fanout import parallel
from common.http_client import get_json, post_json

app = Flask(__name__)

AUTH = os.getenv("AUTH_SERVICE_URL", "http://auth-service:8080")
USER = os.getenv("USER_SERVICE_URL", "http://user-service:8080")
CATALOG = os.getenv("PRODUCT_CATALOG_URL", "http://product-catalog:8080")
INVENTORY = os.getenv("INVENTORY_SERVICE_URL", "http://inventory-service:8080")
PRICING = os.getenv("PRICING_SERVICE_URL", "http://pricing-service:8080")
CART = os.getenv("CART_SERVICE_URL", "http://cart-service:8080")
ORDER = os.getenv("ORDER_SERVICE_URL", "http://order-service:8080")
SEARCH = os.getenv("SEARCH_SERVICE_URL", "http://search-service:8080")
RECOMMEND = os.getenv("RECOMMENDATION_SERVICE_URL", "http://recommendation-service:8080")
FLASH = os.getenv("FLASH_SALE_SERVICE_URL", "http://flash-sale-service:8080")
ANALYTICS = os.getenv("ANALYTICS_SERVICE_URL", "http://analytics-service:8080")


def track(event_type):
    threading.Thread(
        target=post_json,
        args=(f"{ANALYTICS}/events", {"event": event_type}),
        kwargs={"timeout": 2, "name": "analytics"},
        daemon=True,
    ).start()


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "api-gateway"})


@app.route("/api/v1/products")
def list_products():
    track("browse_catalog")
    results = parallel({
        "catalog": lambda: get_json(f"{CATALOG}/products", timeout=8, name="product-catalog"),
        "recommendations": lambda: get_json(f"{RECOMMEND}/recommendations", timeout=5, name="recommendation"),
    })
    return jsonify({"products": results["catalog"], "recommendations": results["recommendations"]})


@app.route("/api/v1/products/<product_id>")
def product_detail(product_id):
    track("view_product")
    results = parallel({
        "catalog": lambda: get_json(f"{CATALOG}/products/{product_id}", timeout=8, name="product-catalog"),
        "inventory": lambda: get_json(f"{INVENTORY}/stock/{product_id}", timeout=6, name="inventory"),
        "pricing": lambda: get_json(f"{PRICING}/price/{product_id}", timeout=5, name="pricing"),
    })
    return jsonify({"product": results["catalog"], "inventory": results["inventory"], "pricing": results["pricing"]})


@app.route("/api/v1/search")
def search():
    track("search")
    query = request.args.get("q", "laptop")
    return jsonify(get_json(f"{SEARCH}/search?q={query}", timeout=8, name="search"))


@app.route("/api/v1/flash-sale")
def flash_sale():
    track("flash_sale_view")
    return jsonify(get_json(f"{FLASH}/active", timeout=10, name="flash-sale"))


@app.route("/api/v1/flash-sale/<product_id>/reserve", methods=["POST"])
def flash_reserve(product_id):
    track("flash_sale_reserve")
    user_id = request.json.get("user_id", "anon") if request.json else "anon"
    results = parallel({
        "auth": lambda: get_json(f"{AUTH}/validate", timeout=5, name="auth"),
        "reservation": lambda: post_json(
            f"{FLASH}/reserve/{product_id}",
            {"user_id": user_id},
            timeout=12,
            name="flash-sale",
        ),
    })
    return jsonify(results)


@app.route("/api/v1/cart", methods=["GET", "POST"])
def cart():
    track("cart")
    body = request.json or {}
    if request.method == "POST":
        results = parallel({
            "auth": lambda: get_json(f"{AUTH}/validate", timeout=5, name="auth"),
            "cart": lambda: post_json(f"{CART}/items", body, timeout=8, name="cart"),
        })
    else:
        results = parallel({
            "auth": lambda: get_json(f"{AUTH}/validate", timeout=5, name="auth"),
            "cart": lambda: get_json(f"{CART}/items", timeout=8, name="cart"),
        })
    return jsonify(results)


@app.route("/api/v1/orders", methods=["POST"])
def place_order():
    track("checkout")
    payload = request.json or {}
    results = parallel({
        "auth": lambda: get_json(f"{AUTH}/validate", timeout=5, name="auth"),
        "user": lambda: get_json(f"{USER}/profile", timeout=5, name="user"),
        "order": lambda: post_json(f"{ORDER}/orders", payload, timeout=20, name="order"),
    })
    return jsonify(results)


@app.route("/")
def home():
    calls = {
        "flash_sale": lambda: get_json(f"{FLASH}/active", timeout=10, name="flash-sale"),
        "featured": lambda: get_json(f"{CATALOG}/featured", timeout=8, name="product-catalog"),
    }
    if random.random() < 0.35:
        calls["recommendations"] = lambda: get_json(
            f"{RECOMMEND}/recommendations", timeout=5, name="recommendation"
        )
    if random.random() < 0.25:
        calls["search"] = lambda: get_json(f"{SEARCH}/search?q=deal", timeout=8, name="search")
    return jsonify(parallel(calls))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
