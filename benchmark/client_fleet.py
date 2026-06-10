#!/usr/bin/env python3
"""
Multi-IP HTTP load generator for Dhrishti client IP visibility.

Traffic to localhost:8080 is proxied by Docker — every client appears as one
bridge IP. This script binds distinct alias IPs on the docker bridge interface
and connects directly to the api-gateway container IP so eBPF sees many clients.

Requires: sudo (ip addr add on bridge), python3

Usage (called automatically by baseline.sh / benchmark.sh):
  sudo CLIENT_IPS=172.21.0.200,172.21.0.201 BRIDGE_IF=br-abc123 \\
    BASE_URL=http://172.21.0.5:8080 python3 client_fleet.py
"""

import os
import random
import signal
import socket
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request

NUM_CLIENTS = int(os.getenv("NUM_CLIENTS", "10"))
RPS_PER_CLIENT = float(os.getenv("RPS_PER_CLIENT", "2"))
DURATION = int(os.getenv("DURATION", "120"))
BASE_URL = os.getenv("BASE_URL", "http://127.0.0.1:8080").rstrip("/")
BRIDGE_IF = os.getenv("BRIDGE_IF", "lo")
IP_PREFIX = os.getenv("CLIENT_IP_PREFIX", "10.254.0")
CLIENT_IPS_RAW = os.getenv("CLIENT_IPS", "")

PATHS = [
    "/api/v1/products",
    "/api/v1/flash-sale",
    "/api/v1/search?q=deal",
    "/api/v1/products/sku-1001",
    "/api/v1/cart",
]

stop = threading.Event()


def client_ips() -> list[str]:
    if CLIENT_IPS_RAW.strip():
        return [ip.strip() for ip in CLIENT_IPS_RAW.split(",") if ip.strip()]
    return [f"{IP_PREFIX}.{i}" for i in range(1, NUM_CLIENTS + 1)]


def add_alias(ip: str) -> None:
    subprocess.run(
        ["ip", "addr", "add", f"{ip}/32", "dev", BRIDGE_IF],
        check=False,
        capture_output=True,
    )


def del_alias(ip: str) -> None:
    subprocess.run(
        ["ip", "addr", "del", f"{ip}/32", "dev", BRIDGE_IF],
        check=False,
        capture_output=True,
    )


def bind_source_ip(source_ip: str):
    """Return a urlopen opener that binds outbound TCP to source_ip."""

    class BoundHTTPConnection(urllib.request.http.client.HTTPConnection):
        def connect(self):
            self.sock = socket.create_connection(
                (self.host, self.port),
                timeout=self.timeout,
                source_address=(source_ip, 0),
            )

    class BoundHTTPHandler(urllib.request.HTTPHandler):
        def http_open(self, req):
            return self.do_open(BoundHTTPConnection, req)

    return urllib.request.build_opener(BoundHTTPHandler())


def worker(client_idx: int, source_ip: str) -> None:
    opener = bind_source_ip(source_ip)
    interval = 1.0 / max(RPS_PER_CLIENT, 0.1)
    ok, fail = 0, 0

    while not stop.is_set():
        path = random.choice(PATHS)
        url = f"{BASE_URL}{path}"
        try:
            with opener.open(url, timeout=15) as resp:
                resp.read(256)
            ok += 1
        except (urllib.error.URLError, TimeoutError, OSError):
            fail += 1
        time.sleep(interval * (0.7 + random.random() * 0.6))

    print(f"  client {client_idx} ({source_ip}): {ok} ok, {fail} fail")


def main() -> int:
    if os.geteuid() != 0:
        print("client_fleet.py needs root:  sudo python3 client_fleet.py")
        return 1

    ips = client_ips()
    if not ips:
        print("No client IPs configured")
        return 1

    print(f"Multi-client fleet: {len(ips)} IPs via {BRIDGE_IF} → {BASE_URL}")
    print(f"  IPs: {ips[0]} .. {ips[-1]}  ({RPS_PER_CLIENT} req/s each, {DURATION}s)")

    for ip in ips:
        add_alias(ip)

    def cleanup(*_):
        stop.set()
        for ip in ips:
            del_alias(ip)

    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)

    threads = [
        threading.Thread(target=worker, args=(i + 1, ip), daemon=True)
        for i, ip in enumerate(ips)
    ]
    for t in threads:
        t.start()

    try:
        time.sleep(DURATION)
    finally:
        cleanup()
        for t in threads:
            t.join(timeout=5)

    print("Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
