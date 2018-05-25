"""
Helpers for uploading with various python versions.
"""

import threading


def parallel(func, args):
    """
    Parallelize upload, using multiple workers, each uploading a range of the
    file.

    Arguments:
        func    upload function using this signature:
                func(filename, offset, size, url, blocksize)
        args    parsed benchmark arguments.
    """
    threads = []
    for i, (offset, size) in enumerate(
            ranges(args.size, args.workers, args.blocksize)):
        t = threading.Thread(
            target=func,
            args=(args.file, offset, size, args.url, args.blocksize),
            name="upload/%d" % i)
        t.daemon = True
        threads.append(t)
        t.start()

    for t in threads:
        t.join()


def ranges(size, count, min_size):
    """
    Generate count ranges tuples (offset, size), covering specified size.

    Ranges are aligned to min_size. If size is less then min_size, there will
    be only one range.

    If size is not a power of min_size, the last range will be smaller.
    """
    chunk = round_up(size // count, min_size)
    pos = 0
    while pos < size:
        yield pos, min(size - pos, chunk)
        pos += chunk


def round_up(n, size):
    """
    Round an integer to next power of size. Size must be power of 2.
    """
    assert size & (size - 1) == 0, "size is not power of 2"
    return ((n - 1) | (size - 1)) + 1


class LimitedReader(object):
    """
    Wrap a file object, limiting the number of bytes read from the file.

    Needed for any python version, since HTTPConnection does not limit the size
    of the upload. Sending extra data after the promised content length will
    cause the next request using the same connection fo fail.
    """

    def __init__(self, reader, size):
        self._reader = reader
        self._limit = size

    def read(self, n=None):
        if n is None:
            n = self._limit
        else:
            n = min(n, self._limit)
        chunk = self._reader.read(n)
        self._limit -= len(chunk)
        return chunk


class BlockIterator(object):
    """
    Iterated over blocks of data from reader.

    Useful with python 3 before python 3.7. If an object does not have a
    read(n) method, HTTPConnection will try to iterate over the object. This
    way we can control the buffersize when uploading files.

    In python 3.7 we can set the HTTPConnection's blocksize instead.
    """

    def __init__(self, reader, blocksize):
        self._reader = reader
        self._blocksize = blocksize

    def __iter__(self):
        while True:
            chunk = self._reader.read(self._blocksize)
            if not chunk:
                return
            yield chunk
