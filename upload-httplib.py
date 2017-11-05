"""
Benchmrk for uploading files using httplib.
"""

import ssl

# Disable certificate verification, not interesitng for this test
ssl._create_default_https_context = ssl._create_unverified_context

import sys

# Python 2/3 compatibility. Don't use six to allow testing with python 3.7
# build that does not have six installed.
try:
    from http import client
except ImportError:
    import httplib as client

import bench

with bench.run() as args:
    conn = client.HTTPSConnection(args.url.netloc)

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
