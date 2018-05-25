import os
import subprocess

SIZE_MB = int(os.environ.get("UPLOAD_SIZE_MB", "1024"))


def upload(variant, size_mb, blocksize_kb, workers=1):
    cmd = [
        "./upload-%s" % variant,
        "--size-mb", str(size_mb),
        "--blocksize-kb", str(blocksize_kb),
    ]

    if workers > 1:
        cmd.extend(("--workers", str(workers)))

    cmd.extend(("/dev/zero", "https://localhost:8000/"))

    p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    if p.returncode != 0:
        raise RuntimeError("Command failed: cmd=%s, rc=%s, out=%r, err=%r"
                           % (cmd, p.returncode, out, err))
    return out
