"""
Common benchmaking utilities.
"""

from __future__ import print_function

import argparse
import os
import time

from contextlib import contextmanager

import six
from six.moves.urllib.parse import urlparse, urlunparse

KIB = 1024
MIB = 1024 * KIB
GIB = 1024 * MIB


@contextmanager
def run():
    args = parse_args()
    start = time.time()
    yield args
    elapsed = time.time() - start
    print("Uploaded %.2f GiB in %.2f seconds (%.2f MiB/s)" % (
          float(args.size) / GIB, elapsed, float(args.size) / MIB / elapsed))


def gibibyte(s):
    return int(s) * GIB


def kibibyte(s):
    return int(s) * KIB


def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument(
        "--size-gb",
        "-s",
        dest="size",
        type=gibibyte,
        help=("upload size in GiB (default file size). Must be specied when "
              "uploading character special file like /dev/zero."))
    parser.add_argument(
        "--blocksize-kb",
        "-b",
        dest="blocksize",
        type=kibibyte,
        default=8192,
        help="block size in KiB (default 8 KiB)")
    parser.add_argument(
        "file",
        help=("file to upload. Can be a file, a block device like /dev/sdb, "
              "or a character special file like /dev/zero."))
    parser.add_argument(
        "url",
        type=urlparse,
        help="upload url. Only https:// supported.")

    args = parser.parse_args()

    if args.size is None:
        # Try to get the file or block device size
        with open(args.file) as f:
            f.seek(0, os.SEEK_END)
            args.size = f.tell()
        if args.size == 0:
            parser.error("Cannot get %r size - please specify --size-gb" % args.file)

    return args


class LimitedFile(object):

    def __init__(self, reader, size, blocksize=8192):
        self._reader = reader
        self._size = size
        self._blocksize = blocksize
        self._limit = size

    if six.PY2:
        def read(self, n=None):
            """
            Readable interface for python 2. Python 3 will use more efficient
            __iter__ if read is not define.
            """
            return self._read(n)

    def __iter__(self):
        """
        Help requests to use streaming. Not used in python 2, but in python 3
        this allows controlling the blocksize.
        """
        while True:
            chunk = self._read(self._blocksize)
            if not chunk:
                break
            yield chunk

    def __len__(self):
        """
        Unlike httplib or go net/http, that do not try to magically get the
        length of the file, and allow the user to set the content-length
        header, requests try various magic behhind your back.  Defining this
        force requests to set a content-length header and use indentity
        transfer encoding.
        """
        return self._size

    def _read(self, n=None):
        if n is None:
            n = self._limit
        else:
            n = min(n, self._limit)
        chunk = self._reader.read(n)
        self._limit -= len(chunk)
        return chunk
