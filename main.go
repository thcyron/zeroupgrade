package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

var (
	listen = flag.String("listen", "", "listen address (tcp)")

	fd  uintptr
	cmd *exec.Cmd

	active      = "a"
	inactive    = "b"
	reloadable  = true
	terminating = false

	exitch chan string
	killch chan struct{}
)

func init() {
	flag.Parse()

	if *listen == "" {
		die("missing listen address")
	}
	if flag.NArg() == 0 {
		die("missing command")
	}

	exitch = make(chan string)
	killch = make(chan struct{})
}

func main() {
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		die("cannot listen: %v", err)
	}

	fdl := getfd(listener)

	// I’m not quite sure why we need to dup the file descriptor here,
	// but doing so fixes situations where the forked process is unable
	// to open the inherited file descriptor.
	dupfd, err := syscall.Dup(int(fdl))
	if err != nil {
		die("dup: %v", err)
	}
	fd = uintptr(dupfd)
	if err := preparefd(fd); err != nil {
		die("could not prepare fd: %v", err)
	}

	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)

	start(active) // Start the very first process.

	for {
		select {
		case <-killch:
			cmd.Process.Signal(syscall.SIGTERM)
		case id := <-exitch:
			if id == active {
				if terminating {
					os.Exit(0)
				}
				die("active process exited")
			} else {
				reloadable = true
			}
		case sig := <-sigch:
			switch sig {
			case syscall.SIGUSR2:
				if reloadable {
					reload()
				}
			case syscall.SIGTERM, syscall.SIGINT:
				terminating = true
				close(sigch)
				cmd.Process.Signal(syscall.SIGTERM)
			}
		}
	}
}

func start(ab string) {
	args := flag.Args()
	c := exec.Command(args[0], args[1:]...)
	c.Env = append(os.Environ(), fmt.Sprintf("LISTENFD=%d", fd))
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		die("starting command for process %s failed: %v", ab, err)
	}

	go func() {
		if err := c.Wait(); err != nil {
			die("command for process %s failed: %v", ab, err)
		}
		exitch <- ab
	}()

	cmd = c
}

func reload() {
	oldCmd := cmd
	start(inactive)
	reloadable = false
	active, inactive = inactive, active

	time.AfterFunc(5*time.Second, func() {
		oldCmd.Process.Signal(syscall.SIGTERM)
	})
}

func getfd(l net.Listener) uintptr {
	fd := reflect.ValueOf(l).Elem().FieldByName("fd").Elem()
	return uintptr(fd.FieldByName("sysfd").Int())
}

func preparefd(fd uintptr) error {
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETFD, 0)
	if errno != 0 {
		return fmt.Errorf("fcntl(F_GETFD): return value %v", errno)
	}
	flags &^= syscall.FD_CLOEXEC
	_, _, errno = syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFD, flags)
	if errno != 0 {
		return fmt.Errorf("fcntl(F_SETFD): return value %v", errno)
	}
	return nil
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("zeroupgrade: %s\n", format), args...)
	os.Exit(1)
}
