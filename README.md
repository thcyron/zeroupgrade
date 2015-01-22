# zeroupgrade

Zeroupgrade upgrades network servers with zero downtime. Unlike nginx
and Unicorn, where the upgrading logic is integrated in the program
itself, zeroupgrade is a separate binary which does the hard work.
Client programs only require minimal changes to allow zero downtime
upgrades.

While zeroupgrade is written in Go, client programs may be written
in any language.

**Be careful. Zeroupgrade hasn’t been used in production yet.**

## Usage

    zeroupgrade -listen localhost:8080 command...

Sending `SIGUSR2` to zeroupgrade performs a zero downtime upgrade.

Sending `SIGTERM` or `SIGINT` shuts down zeroupgrade and its
child processes.

## Client program modifications

The only thing client programs must do in order to support zero
downtime upgrades is to listen on a specific file descriptor.
The file descreiptor is given in the `LISTENFD` environment variable.

In Go, it looks like this:

```go
if listenfd := os.Getenv("LISTENFD"); listenfd != "" {
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
- Race tests
