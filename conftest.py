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
