"""
Benchmrk for uploading files using requests.
"""

from __future__ import print_function

import sys
import time

import requests
import util

BLOCK_SIZE = 512 * 1024
MB = 1024**2
GB = 1024**3

size = int(sys.argv[1]) * GB
url = sys.argv[2]

start = time.time()

# verify=False to disable certificate verification.
with open("/dev/zero", "rb") as f:
    f = util.LimitedReader(f, size)
    r = requests.put(url, data=f, verify=False)

elapsed = time.time() - start

assert r.status_code == 200

print("Uploaded %.2fg in %.2f seconds (%.2fm/s)" % (
    float(size) / GB, elapsed, float(size) / MB / elapsed))
