import requests
from flask import Flask

app = Flask(__name__)

"""
Order service.

Creates multi-dependency fanout:

order
 ├── inventory
 └── payment

Also introduces retry behavior.
"""


@app.route("/order")
def order():

    inventory = requests.get(
        "http://inventory-service:8080/inventory",
        headers={"Connection": "close"},
        timeout=2,
    )

    payment_response = None

    # retry payment once
    for _ in range(2):
        payment = requests.get(
            "http://payment-service:8080/pay",
            headers={"Connection": "close"},
            timeout=3,
        )

        payment_response = payment.json()

        if payment.status_code == 200:
            break

        # retry delay
        time.sleep(0.3)

    return {
        "inventory": inventory.json(),
        "payment": payment_response,
    }


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
