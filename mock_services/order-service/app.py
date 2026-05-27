import time

import requests
from flask import Flask, jsonify

app = Flask(__name__)

"""
Order fans out to inventory then payment (with one retry).
Sequential slow downstream calls keep multiple edges active at once.
"""


@app.route("/order")
def order():
    result = {"inventory": None, "payment": None}

    try:
        inventory = requests.get(
            "http://inventory-service:8080/inventory",
            timeout=5,
        )
        result["inventory"] = {
            "status": inventory.status_code,
            "body": inventory.json(),
        }
    except requests.exceptions.RequestException as e:
        result["inventory"] = {"status": 502, "error": str(e)}

    payment_response = None
    for _ in range(2):
        try:
            payment = requests.get(
                "http://payment-service:8080/pay",
                timeout=5,
            )
            payment_response = {
                "status": payment.status_code,
                "body": payment.json(),
            }
            if payment.status_code == 200:
                break
        except requests.exceptions.RequestException as e:
            payment_response = {"status": 502, "error": str(e)}

        time.sleep(0.2)

    result["payment"] = payment_response
    return jsonify(result)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
