import random
import time

import requests

"""
Traffic generator — burst + idle.

Each gateway request holds client→gateway open for several seconds while
downstream services sleep (auth/inventory/payment). That makes active_connections
visible on the graph between WebSocket snapshots.
"""

GATEWAY_URL = "http://gateway:8080/"

while True:
    idle_seconds = random.uniform(15, 25)
    print(f"[client] idle {idle_seconds:.1f}s")
    time.sleep(idle_seconds)

    burst_size = random.randint(1, 2)
    print(f"[client] burst x{burst_size}")

    for i in range(burst_size):
        try:
            response = requests.get(GATEWAY_URL, timeout=30)
            print(f"  [{i + 1}/{burst_size}] status={response.status_code}")
        except Exception as e:
            print(f"  [{i + 1}/{burst_size}] failed:", e)

        if i < burst_size - 1:
            time.sleep(random.uniform(1.0, 2.0))
