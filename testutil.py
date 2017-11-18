import os
import subprocess
import threading
import time

SIZE_MB = int(os.environ.get("UPLOAD_SIZE_MB", "1024"))
MAX_WORKERS = int(os.environ.get("MAX_WORKERS", "0"))


def upload_parallel(variant, size_mb, blocksize_kb):
    workers = os.sysconf("SC_NPROCESSORS_ONLN")

    if MAX_WORKERS:
        workers = min(workers, MAX_WORKERS)

    if size_mb < workers:
        size_mb = workers

    size_per_worker = size_mb // workers

    def run():
        upload(variant, size_per_worker, blocksize_kb)

    start = time.time()

    threads = []
    try:
        for i in range(workers):
            t = threading.Thread(target=run, name="upload/%d" % i)
            t.daemon = True
            t.start()
            threads.append(t)

    finally:
        for t in threads:
            t.join()

    elapsed = time.time() - start

    msg = "Uploaded %.2f MiB in %.2f seconds using %d workers (%.2f MiB/s)"
    return msg % (size_mb, elapsed, workers, size_mb / elapsed)


def upload(variant, size_mb, blocksize_kb):
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
