from http.server import BaseHTTPRequestHandler, HTTPServer

HOST = "0.0.0.0"
PORT = 8080


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        print(f"request received from {self.client_address}")

        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"Hello, server!")


server = HTTPServer((HOST, PORT), Handler)
print(f"server listening on {HOST}:{PORT}")
server.serve_forever()
