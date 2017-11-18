# http-bench

[![Build Status](https://travis-ci.org/nirs/http-bench.svg?branch=master)](https://travis-ci.org/nirs/http-bench)

Benchmark tools for python HTTP I/O. Was written for optimizing
[ovirt-imageio](https://github.com/ovirt/ovirt-imageio), and
[ovirt-engine-sdk](https://github.com/ovirt/ovirt-engine-sdk).


## Setup

Install requirements:

    $ pip install -r requirements.txt


### Installing go

To use the go server or client, you need to install the golang and make
packages

On Fedora:

    $ sudo dnf install golang make


## Building

    $ make


## Running the throughput tests

To run the tests with all install python versions:

    $ make test

The default test use upload size of 1024 MiB, which takes couple of
seconds to upload. If you want to run the tests quickly, you can specify
a smaller upload size using environment variable:

    UPLOAD_SIZE_MB=1 make test

The parallel uploads tests start one worker per cpu. If you want to
limit the number of workers, use:

    MAX_WORKERS=2 make test

To run with specific python version:

    $ tox -e py27


## Measuring upload throughput

Start the go server:

    $ ./serve

To disable HTTP/2:

    $ GODEBUG=http2server=0 ./serve

You can also use the python server for somewhat lower results:

    $ python serve.py

Run upload tests. This example uploads filename to the server using
python httplib:

    $ python upload-httplib.py filename http://localhost:8000/

You can upload entire block device:

    $ python upload-httplib.py /dev/sdb http://localhost:8000/

Or a character special file like /dev/zero - in this case you must
specify the size of the upload:

    $ python upload-httplib.py --size-gb 10 /dev/zero http://localhost:8000/

To test how block size effects the throughput:

    $ python upload-httplib.py --blocksize-kb 512 --size-gb 10 /dev/zero http://localhost:8000/


### Tests

- upload-httplib.py - using httplib (http.client on python 3)
- upload-requests.py - using the requests library
- upload.go - go version, using HTTP/2 or HTTP/1.1. This tool ignores
  the --blocksize-kb option since go uses hardcoded value of 4kb.
