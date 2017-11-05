"""
Common benchmaking utilities.
"""

from __future__ import print_function

import argparse
import os
import time

from contextlib import contextmanager

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
        "--buffer-size-kb",
        "-b",
        dest="buffer_size",
        type=kibibyte,
        default=8192,
        help="buffer size in KiB (default 8)")
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
