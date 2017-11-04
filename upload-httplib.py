"""
Benchmrk for uploading files using httplib.
"""

from __future__ import print_function

import ssl
import sys
import time

import util

# Python 2/3 compatibility. Don't use six to allow testing with python 3.7
# build that does not have six installed.
try:
    from http import client
except ImportError:
    import httplib as client

try:
    from urllib.parse import urlparse
except ImportError:
    from urlparse import urlparse

# Disable certificate verification, not interesitng for this test
ssl._create_default_https_context = ssl._create_unverified_context

BLOCK_SIZE = 512 * 1024
MB = 1024**2
GB = 1024**3

size = int(sys.argv[1]) * GB
url = urlparse(sys.argv[2])

start = time.time()

conn = client.HTTPSConnection(url.netloc)

# TODO: link to patch adding this.
if hasattr(conn, "blocksize"):
    conn.blocksize = BLOCK_SIZE

conn.putrequest("PUT", url.path)
conn.putheader("Content-Length", "%d" % (size,))
conn.endheaders()

with open("/dev/zero", "rb") as f:
    f = util.LimitedReader(f, size)
    conn.send(f)

resp = conn.getresponse()

elapsed = time.time() - start

assert resp.status == 200

print("Uploaded %.2fg in %.2f seconds (%.2fm/s)" % (
      float(size) / GB, elapsed, float(size) / MB / elapsed))
