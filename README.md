# srcsnk

A simple server that sends and receives arbitrary amounts of data at arbitrary
speeds. Useful for testing proxies and clients.

## Installation

    go get -u github.com/bluekeyes/srcsnk

## Usage

Run `srcsnk` to start the server. Once the server is running, you can download
a file at any path:

    $ curl localhost:8000/an/arbitrary/file.bin?size=10M -o file.bin

Or upload a file to any path:

    $ curl -T file.bin localhost:8000/an/arbitrary/path/file.bin

All the received data is discarded by the server.

Each endpoint accepts several paramters described below.

### Paramters

- `size` - (download only) controls the size of the downloaded file. It takes
  the suffixes `B`, `K`, `M`, and `G` for bytes, kilobytes, megabytes, or
  gigabytes, respectively; these have the same meaning as in the `dd` command.

- `rate` - controls the download or upload rate, in bytes per second. It
  accepts the same suffixes as the `size` paramter.

- `delayPre` - the amount of time to wait before processing the request,
  including reading the request body. It accepts any value allowed by
  [`time.ParseDuration`][].

- `delayRes` - the amount of time to wait before sending the initial response
  headers. It accepts any value allowed by [`time.ParseDuration`][]. For
  downloads, `delayPre` and `delayRes` are interchangeable.

[`time.ParseDuration`]: https://golang.org/pkg/time/#ParseDuration

### Flags

The command accepts the following flags:

- `-address` - sets the address (IP and port) the server listens on; defaults to `127.0.0.1:8000`

- `-log-file` - sets the file where log output is written; default to standard out
