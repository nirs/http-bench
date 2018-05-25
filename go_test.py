import testutil

# Go uses hardcoded value, not way to change the blocksize.
BLOCKSIZE_KB = 4


def test_go(server):
    print(testutil.upload("go", testutil.SIZE_MB, BLOCKSIZE_KB))
