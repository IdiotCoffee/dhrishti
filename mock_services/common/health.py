from flask import jsonify


def health_response(service_name):
    return jsonify({"status": "ok", "service": service_name})
