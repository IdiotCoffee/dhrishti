import random
import time

import requests

"""
Synthetic traffic generator.

This continuously drives the system.

Important:
observability systems need workload.
Without runtime behavior:
there is nothing to observe.
"""


while True:
    try:
        response = requests.get(
            "http://gateway:8080/",
            headers={"Connection": "close"},
            timeout=5,
        )

        print(response.json())

    except Exception as e:
        print("request failed:", e)

    # variable request spacing
    time.sleep(random.uniform(1, 3))
