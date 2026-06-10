import threading

import requests
from requests.adapters import HTTPAdapter

_local = threading.local()


def _session():
    if not hasattr(_local, "session"):
        session = requests.Session()
        adapter = HTTPAdapter(pool_connections=32, pool_maxsize=64, max_retries=0)
        session.mount("http://", adapter)
        session.mount("https://", adapter)
        _local.session = session
    return _local.session


def get_json(url, timeout=5, name="service"):
    try:
        resp = _session().get(url, timeout=timeout)
        try:
            body = resp.json()
        except ValueError:
            body = {"raw": resp.text[:200]}
        return {"status": resp.status_code, "body": body}
    except requests.exceptions.Timeout:
        return {"status": 504, "error": f"{name} timed out"}
    except requests.exceptions.RequestException as exc:
        return {"status": 502, "error": f"{name} unreachable: {exc}"}


def post_json(url, payload=None, timeout=5, name="service"):
    try:
        resp = _session().post(url, json=payload or {}, timeout=timeout)
        try:
            body = resp.json()
        except ValueError:
            body = {"raw": resp.text[:200]}
        return {"status": resp.status_code, "body": body}
    except requests.exceptions.Timeout:
        return {"status": 504, "error": f"{name} timed out"}
    except requests.exceptions.RequestException as exc:
        return {"status": 502, "error": f"{name} unreachable: {exc}"}
