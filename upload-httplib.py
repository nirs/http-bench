"""
Benchmrk for uploading files using httplib.
"""

import ssl

# Disable certificate verification, not interesitng for this test
ssl._create_default_https_context = ssl._create_unverified_context

import sys

from six.moves import http_client

import bench

with bench.run() as args:
    conn = http_client.HTTPSConnection(args.url.netloc)

    with open(args.file, "rb") as f:
        f = bench.LimitedFile(f, args.size, args.buffer_size)
        conn.request("PUT", args.url.path, body=f,
                     headers={"Content-Length": str(args.size)})

    resp = conn.getresponse()
    assert resp.status == 200
