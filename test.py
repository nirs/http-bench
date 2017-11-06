import subprocess
import threading

import pytest


@pytest.fixture(scope="session")
def server():
    server = subprocess.Popen(["./serve"], stdout=subprocess.PIPE)
    try:
        t = threading.Thread(target=server.communicate)
        t.daemon = True
        t.start()
        yield
    finally:
        server.kill()
        server.wait()
        t.join()


@pytest.mark.parametrize("mod", ["httplib", "requests"])
@pytest.mark.parametrize("blocksize",
    [8, 32, 64, 128, 256, 512, 1024, 2048, 4096])
def test_single_upload(server, mod, blocksize):
    print(upload(mod, blocksize))


def upload(mod, blocksize_kb):
    cmd = [
        "python",
        "upload-%s.py" % mod,
        "--size-gb", "1",
        "--blocksize-kb", str(blocksize_kb),
        "/dev/zero", "https://localhost:8000/"
    ]
    p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    if p.returncode != 0:
        raise RuntimeError("Command failed: cmd=%s, rc=%s, out=%r, err=%r"
                           % (cmd, rc, out, err))
    return out
