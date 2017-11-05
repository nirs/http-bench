"""
Benchmrk for uploading files using requests.
"""

import requests
import bench

with bench.run() as args:
    with open(args.file, "rb") as f:
        f = bench.LimitedReader(f, args.size)
        url = bench.urlunparse(args.url)

        # verify=False to disable certificate verification.
        resp = requests.put(url, data=f, verify=False)

    assert resp.status_code == 200
