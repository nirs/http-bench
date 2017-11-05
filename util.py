"""
Common utils
"""

class LimitedReader(object):

    def __init__(self, reader, size):
        self._read = reader.read
        self._size = size
        self._limit = size

    # httplib requires a reader (object with a read(n) method)

    def read(self, n=None):
        if n is None:
            n = self._limit
        else:
            n = min(n, self._limit)
        chunk = self._read(n)
        self._limit -= len(chunk)
        return chunk

    # Ugly hacks for the requests library

    def __iter__(self):
        """
        Fake iterator to fool requests to pass this file-like object to the
        underlying connection. The underlying connection only care about the
        read() method.
        """
        raise NotImplemented

    def __len__(self):
        """
        Unlike httplib or go net/http, that do not try to magically get the
        length of the file, and allow the user to set the content-length
        header, requests try various magic behhind your back.  Defining this
        force requests to set a content-length header.
        """
        return self._size
