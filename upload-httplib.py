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

    # See XXX for the patch adding this
    conn.blocksize = args.buffer_size

    conn.putrequest("PUT", args.url.path)
    conn.putheader("Content-Length", "%d" % (args.size,))
    conn.endheaders()

    with open(args.file, "rb") as f:
        f = bench.LimitedReader(f, args.size)
        conn.send(f)

    resp = conn.getresponse()
    assert resp.status == 200
