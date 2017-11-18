"""
Common benchmaking utilities.
"""

from __future__ import print_function

import argparse
import os
import time

from contextlib import contextmanager

from six.moves.urllib.parse import urlparse

KIB = 1024
MIB = 1024 * KIB


@contextmanager
def run():
    args = parse_args()
    start = time.time()
    yield args
    elapsed = time.time() - start
    print("Uploaded %.2f MiB in %.2f seconds (%.2f MiB/s)" % (
          float(args.size) / MIB, elapsed, float(args.size) / MIB / elapsed))


def megabyte(s):
    return int(s) * MIB


def kibibyte(s):
    return int(s) * KIB


def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument(
        "--size-mb",
        "-s",
        dest="size",
        type=megabyte,
        help=("upload size in MiB (default file size). Must be specied when "
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
            parser.error("Cannot get %r size - please specify --size-mb"
                         % args.file)

    return args
