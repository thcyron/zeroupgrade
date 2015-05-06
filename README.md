# zeroupgrade

Zeroupgrade upgrades network servers with zero downtime. Unlike nginx
and Unicorn, where the upgrading logic is integrated in the program
itself, zeroupgrade is a separate binary which does the hard work.
Client programs only require minimal changes to allow zero downtime
upgrades.

While zeroupgrade is written in Go, client programs may be written
in any language.

## Usage

    zeroupgrade -listen localhost:8080 command...

`-listen addr` may be specified multiple times.

Sending `SIGUSR2` to zeroupgrade performs a zero downtime upgrade.

Sending `SIGTERM` or `SIGINT` shuts down zeroupgrade and its
child processes.

## Client program modifications

The only thing client programs must do in order to support zero
downtime upgrades is to listen on specific file descriptors. The
file descriptors (one for each listen address) are given in the
`ZEROUPGRADE_FDn` environment variables; the file descriptor for
the first listen address being in `ZEROUPGRADE_FD0`, the second in
`ZEROUPGRADE_FD1`, and so on.

In Go, it looks like this:

```go
if listenfd := os.Getenv("ZEROUPGRADE_FD0"); listenfd != "" {
        n, err := strconv.ParseUint(listenfd, 10, 64)
        if err != nil {
                panic(err)
        }
        fd := uintptr(n)
        listener, err = net.FileListener(os.NewFile(fd, ""))
        if err != nil {
                panic(err)
        }
}

http.Serve(listener, nil)
```

## TODO

- Write pidfile
