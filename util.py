"""
Helpers for uploading with various python versions.
"""


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
