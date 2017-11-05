# http-bench

Benchmark tools for python HTTP I/O. Was written for optimizing
[ovirt-imageio](https://github.com/ovirt/ovirt-imageio), and
[ovirt-engine-sdk](https://github.com/ovirt/ovirt-engine-sdk).


## Setup

Create a virtual environment for every python version you want to test:

### Python 2

    $ pip install virtualenv
    $ virtualenv py/27
    $ source py/27/bin/activate
    $ pip install -r requirements.txt

## Python 3

    $ python3.5 -m venv py/35
    $ source py/27/bin/activate
    $ pip install -r requirements.txt


### Installing go

To use the go server or client, you need to install the golang and make
packages

On Fedora:

    $ sudo dnf install golang make


## Building

    $ make


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

To test how buffer size effects the throughput, you need to build
python 3.7 with this patch:
https://github.com/python/cpython/pull/4279

This example uses buffer size of 512 KiB:

    $ python upload-httplib.py --buffer-size-kb 512 --size-gb 10 /dev/zero http://localhost:8000/


### Tests

- upload-httplib.py - using httlib (http.client on python 3)
- upload-requests.py - using the requests library
- upload.go - go version, using HTTP/2 or HTTP/1.1 (always upload from /dev/zero)
