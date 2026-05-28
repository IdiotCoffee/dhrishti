import random
import time
from concurrent.futures import ThreadPoolExecutor

import requests

"""
Traffic generator — burst + idle.

Each gateway request holds client→gateway open for several seconds while
downstream services sleep (auth/inventory/payment). That makes active_connections
visible on the graph between WebSocket snapshots.
"""

GATEWAY_URL = "http://gateway:8080/"

def fire_request(seq):
    try:
        response = requests.get(GATEWAY_URL, timeout=12)
        print(f"  [{seq}] status={response.status_code}")
    except Exception as e:
        print(f"  [{seq}] failed: {e}")


next_fire = time.monotonic()
request_seq = 0

with ThreadPoolExecutor(max_workers=2) as pool:
    while True:
        idle_seconds = random.uniform(5, 7)
        next_fire += idle_seconds

        sleep_for = next_fire - time.monotonic()
        if sleep_for > 0:
            print(f"[client] idle {sleep_for:.1f}s")
            time.sleep(sleep_for)

        burst_size = random.randint(1, 2)
        print(f"[client] burst x{burst_size}")

        for _ in range(burst_size):
            request_seq += 1
            pool.submit(fire_request, request_seq)
            time.sleep(random.uniform(0.2, 0.6))
