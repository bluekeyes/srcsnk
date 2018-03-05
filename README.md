# srcsnk

A simple server that sends and receives arbitrary amounts of data at arbitrary
speeds. Useful for testing proxies and clients.

## Installation

    go get -u github.com/bluekeyes/srcsnk

## Usage

Run `srcsnk` to start the server. The `-address` flag sets the listen address.

Once the server is running, you can download a file at any path:

    $ curl localhost:8000/an/arbitrary/file.bin?size=10M -o file.bin

The `size` parameter controls the size of the downloaded file. It takes the
suffixes `B`, `K`, `M`, or `G` for bytes, kilobytes, megabytes, or gigabytes,
respectively (these have the same meaning as in the `dd` command.)

You can also upload a file to any path:

    $ curl -T file.bin localhost:8000/an/arbitrary/path/

All the received data is discarded by the server.
