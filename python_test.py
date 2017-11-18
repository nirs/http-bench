import pytest
import testutil


@pytest.mark.parametrize("variant", ["httplib", "requests"])
@pytest.mark.parametrize("blocksize", [
    8, 32, 64, 128, 256, 512, 1024, 2048, 4096
])
def test_python(server, variant, blocksize):
    print(testutil.upload(variant, testutil.SIZE_MB, blocksize))


@pytest.mark.parametrize("variant", ["httplib", "requests"])
@pytest.mark.parametrize("blocksize", [32, 64, 128, 256, 512])
def test_python_parallel(server, variant, blocksize):
    print(testutil.upload_parallel(variant, testutil.SIZE_MB, blocksize))
