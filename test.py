import os
import subprocess
import threading
import time

import pytest

SIZE_MB = int(os.environ.get("UPLOAD_SIZE_MB", "1024"))


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


@pytest.mark.parametrize("variant", ["httplib", "requests"])
@pytest.mark.parametrize("blocksize",
    [8, 32, 64, 128, 256, 512, 1024, 2048, 4096])
def test_python(server, variant, blocksize):
    print(upload(variant, blocksize))


def test_go(server):
    # Go uses hardcoded value, not way to change the blocksize.
    print(upload("go", 4))


@pytest.mark.parametrize("variant", ["httplib", "requests"])
@pytest.mark.parametrize("blocksize", [32, 64, 128, 256, 512])
def test_python_parallel(server, variant, blocksize):
    print(upload_parallel(variant, blocksize))


def test_go_parallel(server):
    print(upload_parallel("go", 4))


def upload_parallel(variant, blocksize_kb):
    cpu_count = os.sysconf("SC_NPROCESSORS_ONLN")

    size = SIZE_MB
    if size < cpu_count:
        size = cpu_count

    size_per_worker = size // cpu_count

    def run():
        upload(variant, blocksize_kb, size_mb=size_per_worker)

    start = time.time()

    threads = []
    try:
        for i in range(cpu_count):
            t = threading.Thread(target=run, name="upload/%d" % i)
            t.daemon = True
            t.start()
            threads.append(t)

    finally:
        for t in threads:
            t.join()

    elapsed = time.time() - start

    return "Uploaded %.2f MiB in %.2f seconds using %d workers (%.2f MiB/s)" % (
        size, elapsed, cpu_count, size / elapsed)


def upload(variant, blocksize_kb, size_mb=SIZE_MB):
    cmd = [
        "./upload-%s" % variant,
        "--size-mb", str(size_mb),
        "--blocksize-kb", str(blocksize_kb),
        "/dev/zero", "https://localhost:8000/"
    ]
    p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    if p.returncode != 0:
        raise RuntimeError("Command failed: cmd=%s, rc=%s, out=%r, err=%r"
                           % (cmd, p.returncode, out, err))
    return out
