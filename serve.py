"""
Benchmarking server for measuring upload throughput

This server acccept a PUT request and read the data according to the
Content-Range header, dropping the data.

This is useful for tuning upload code.
"""

import BaseHTTPServer
import ssl

BUF_SIZE = 128 * 1024


class Server(BaseHTTPServer.HTTPServer):

    protocol_version = "HTTP/1.1"


class Handler(BaseHTTPServer.BaseHTTPRequestHandler):

    def do_PUT(self):
        print self.headers
        size = int(self.headers["content-length"])
        pos = 0
        while pos < size:
            to_read = min(size - pos, BUF_SIZE)
            chunk = self.rfile.read(to_read)
            if not chunk:
                self.send_error(400)
                return
            pos += len(chunk)

        self.send_response(200)
        self.send_header("Content-Length", "0")
        self.end_headers()


server = Server(('', 8000), Handler)

server.socket = ssl.wrap_socket(
    server.socket,
    keyfile="key.pem",
    certfile="cert.pem",
    server_side=True)

server.serve_forever()
