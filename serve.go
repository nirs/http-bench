package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"
)

const (
	MB = 1 << 20
)

var (
	blocksize = flag.Int64(
		"blocksize",
		1024*1024,
		`block size for copying data to storage.`)
	output = flag.String(
		"output",
		"/dev/null",
		`output file name; if not set output will be discarded.`)
)

func main() {
	flag.Parse()
	fmt.Printf("Using blocksize=%d\n", *blocksize)
	fmt.Printf("Using output=%s\n", *output)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServeTLS(":8000", "cert.pem", "key.pem", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	logEvent(r, "START", "(%.2f MiB)", float64(r.ContentLength)/float64(MB))

	if r.Method != "PUT" {
		fail(w, r, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == -1 {
		fail(w, r, "Content-Length required", http.StatusBadRequest)
		return
	}

	start := time.Now()

	if _, err := write(r); err != nil {
		fail(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	elapsed := time.Since(start).Seconds()

	logEvent(r, "FINISH", "(%.2f MiB in %.2f seconds, %.2f MiB/s)",
		float64(r.ContentLength)/float64(MB),
		elapsed,
		float64(r.ContentLength)/float64(MB)/elapsed)
}

func fail(w http.ResponseWriter, r *http.Request, msg string, code int) {
	logEvent(r, "ERROR", msg)
	http.Error(w, msg, code)
}

func logEvent(r *http.Request, event string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s %s %q: %s", r.RemoteAddr, event, r.Method, r.URL.Path, message)
}

func write(r *http.Request) (n int64, err error) {
	buf := make([]byte, *blocksize)
	reader := io.LimitReader(r.Body, r.ContentLength)

	file, err := os.OpenFile(*output, os.O_WRONLY, 0)
	if err != nil {
		return 0, err
	}

	if n, err = io.CopyBuffer(file, reader, buf); err != nil {
		return n, err
	}

	if err = file.Sync(); err != nil {
		if errno(err) == syscall.EINVAL {
			// Sync to /dev/null fails with EINVAL; ignore it
			err = nil
		} else {
			fmt.Printf("%T %#v\n", err, err)
			return n, err
		}
	}

	if err = file.Close(); err != nil {
		return n, err
	}

	return n, nil
}

// errno unwraps syscall.Errno from wrapped errors.
func errno(e error) syscall.Errno {
	switch v := e.(type) {
	case *os.PathError:
		return errno(v.Err)
	case *os.SyscallError:
		return errno(v.Err)
	case syscall.Errno:
		return v
	default:
		return 0
	}
}
