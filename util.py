"""
Common utils
"""

class LimitedReader(object):

    def __init__(self, reader, size):
        self._read = reader.read
        self._size = size
        self._limit = size

    def read(self, n=None):
        if n is None:
            n = self._limit
        else:
            n = min(n, self._limit)
        chunk = self._read(n)
        self._limit -= len(chunk)
        return chunk

    def __iter__(self):
        """
        Fake iterator to fool requests that it can pass this to the underlying
        connection, that case only about read().
        """
        raise NotImplemented

    def __len__(self):
        """
        Used by requests to set the content-length

        Need because the server does not support chunked encoding, and reqires
        content-length.
        """
        return self._size
