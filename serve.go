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
	"unsafe"
)

const (
	KB = 1 << 10
	MB = 1 << 20
)

var (
	blocksizeKB = flag.Int(
		"blocksize-kb",
		1024,
		`block size for copying data to storage.`)
	poolsize = flag.Int(
		"poolsize",
		2,
		`number of buffers.`)
	direct = flag.Bool(
		"direct",
		false,
		"use direct I/O")
	output = flag.String(
		"output",
		"/dev/null",
		`output file name; if not set output will be discarded.`)
	debug = flag.Bool(
		"debug",
		false,
		`enable debug logging.`)
)

func main() {
	flag.Parse()

	fmt.Printf("Using blocksizeKB=%d\n", *blocksizeKB)
	fmt.Printf("Using poolsize=%d\n", *poolsize)
	fmt.Printf("Using direct=%v\n", *direct)
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
	flags := os.O_RDWR
	if *direct {
		flags |= syscall.O_DIRECT
	}
	file, err := os.OpenFile(*output, flags, 0)
	if err != nil {
		return 0, err
	}

	if n, err = copyData(file, r.Body); err != nil {
		return n, err
	}

	start := time.Now()
	if err = file.Sync(); err != nil {
		if errno(err) == syscall.EINVAL {
			// Sync to /dev/null fails with EINVAL; ignore it
			err = nil
		} else {
			fmt.Printf("%T %#v\n", err, err)
			return n, err
		}
	}
	if *debug {
		log.Printf("Synced in %.6f seconds\n", time.Since(start).Seconds())
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

type data struct {
	buf []byte
	len int
}

type result struct {
	written int64
	err     error
}

func copyData(dst io.Writer, src io.Reader) (written int64, err error) {
	pool := make(chan []byte, *poolsize)
	work := make(chan *data, *poolsize)
	done := make(chan *result)

	for i := 0; i < *poolsize; i++ {
		pool <- alignedBuffer(*blocksizeKB*KB, 512)
	}

	go writer(dst, work, pool, done)

	for {
		buf := <-pool
		start := time.Now()
		nr, er := io.ReadFull(src, buf)
		if *debug {
			log.Printf("Read %d bytes in %.6f seconds\n", nr, time.Since(start).Seconds())
		}
		if nr > 0 {
			work <- &data{buf: buf, len: nr}
		}
		if er != nil {
			// Getting less bytes or no bytes means the body is consumed.
			if er != io.EOF && er != io.ErrUnexpectedEOF {
				err = er
			}
			break
		}
	}

	close(work)
	r := <-done

	if err != nil {
		return r.written, err
	} else {
		return r.written, r.err
	}
}

func writer(dst io.Writer, work chan *data, pool chan []byte, done chan *result) {
	var written int64
	var err error

	for w := range work {
		start := time.Now()
		nw, err := dst.Write(w.buf[0:w.len])
		if *debug {
			log.Printf("Wrote %d bytes in %.6f seconds\n", nw, time.Since(start).Seconds())
		}
		if nw > 0 {
			written += int64(nw)
		}
		if err != nil {
			break
		}
		if w.len != nw {
			err = io.ErrShortWrite
			break
		}

		pool <- w.buf
	}

	done <- &result{written, err}
}

func alignedBuffer(size int, align int) []byte {
	buf := make([]byte, size+align)
	offset := 0
	remainder := int(uintptr(unsafe.Pointer(&buf[0])) & uintptr(align-1))
	if remainder != 0 {
		offset = align - remainder
	}
	return buf[offset : offset+size]
}
