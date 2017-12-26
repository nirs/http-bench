package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
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
	limitread = flag.Int(
		"limit-read-mbps",
		0,
		"limit read rate in megabytes per seconds.")
	limitwrite = flag.Int(
		"limit-write-mbps",
		0,
		"limit write rate in megabytes per seconds.")
	direct = flag.Bool(
		"direct",
		false,
		"use direct I/O")
	output = flag.String(
		"output",
		"/dev/null",
		`output file name; if not set output will be discarded.`)
	stats = flag.Bool(
		"stats",
		false,
		"show upload stats")
	debug = flag.Bool(
		"debug",
		false,
		`enable debug logging.`)

	measureRead  = false
	measureWrite = false
)

func main() {
	flag.Parse()

	fmt.Printf("Using blocksizeKB=%v\n", *blocksizeKB)
	fmt.Printf("Using poolsize=%v\n", *poolsize)
	fmt.Printf("Using limitread=%v\n", *limitread)
	fmt.Printf("Using limitwrite=%v\n", *limitwrite)
	fmt.Printf("Using direct=%v\n", *direct)
	fmt.Printf("Using output=%v\n", *output)
	fmt.Printf("Using stats=%v\n", *stats)
	fmt.Printf("Using debug=%v\n", *debug)

	measureRead = *stats || *debug || (*limitread != 0)
	measureWrite = *stats || *debug || (*limitwrite != 0)

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

	clock := newClock()
	clock.Start("total")

	if _, err := write(r, clock); err != nil {
		fail(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	elapsed := clock.Stop("total").Seconds()

	if *stats {
		log.Printf("Stats: %v", clock)
	}

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

func write(r *http.Request, clock *Clock) (n int64, err error) {
	flags := os.O_RDWR
	if *direct {
		flags |= syscall.O_DIRECT
	}
	file, err := os.OpenFile(*output, flags, 0)
	if err != nil {
		return 0, err
	}

	clock.Start("copy")
	if n, err = copyData(file, r.Body, clock); err != nil {
		return n, err
	}
	clock.Stop("copy")

	clock.Start("sync")
	if err = file.Sync(); err != nil {
		if errno(err) == syscall.EINVAL {
			// Sync to /dev/null fails with EINVAL; ignore it
			err = nil
		} else {
			return n, err
		}
	}
	elapsed := clock.Stop("sync")
	if *debug {
		log.Printf("Synced in %.6f seconds\n", elapsed.Seconds())
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

func copyData(dst io.Writer, src io.Reader, clock *Clock) (written int64, err error) {
	pool := make(chan []byte, *poolsize)
	work := make(chan *data, *poolsize)
	done := make(chan *result)

	for i := 0; i < *poolsize; i++ {
		pool <- alignedBuffer(*blocksizeKB*KB, 512)
	}

	go writer(dst, work, pool, done, clock)

	for {
		buf := <-pool
		if measureRead {
			clock.Start("read")
		}
		nr, er := io.ReadFull(src, buf)
		if measureRead {
			if *limitread > 0 {
				limitRate(nr, clock.Elapsed("read"), *limitread)
			}
			elapsed := clock.Stop("read")
			if *debug {
				log.Printf("Read %d bytes in %.6f seconds\n", nr, elapsed.Seconds())
			}
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

func writer(dst io.Writer, work chan *data, pool chan []byte, done chan *result, clock *Clock) {
	var written int64
	var err error

	for w := range work {
		if measureWrite {
			clock.Start("write")
		}
		nw, err := dst.Write(w.buf[0:w.len])
		if measureWrite {
			if *limitwrite > 0 {
				limitRate(nw, clock.Elapsed("write"), *limitwrite)
			}
			elapsed := clock.Stop("write")
			if *debug {
				log.Printf("Wrote %d bytes in %.6f seconds\n", nw, elapsed.Seconds())
			}
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

// limitRate limit operation rate by sleeping until the expected time.
// TODO: sleep little less time, so time.Since(start) returns the expected value.
func limitRate(n int, elapsed time.Duration, rate int) {
	expected := time.Duration(float64(n) / float64(MB) / float64(rate) * 1e09)
	if expected > elapsed {
		time.Sleep(expected - elapsed)
	}
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

type Timer struct {
	total   time.Duration
	started time.Time
	running bool
}

type Clock struct {
	mutex  sync.Mutex
	timers map[string]*Timer
	names  []string
}

func newClock() *Clock {
	return &Clock{
		timers: map[string]*Timer{},
		names:  []string{},
	}
}

func (c *Clock) Start(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t, ok := c.timers[name]
	if !ok {
		t = &Timer{}
		c.timers[name] = t
		c.names = append(c.names, name)
	} else {
		if t.running {
			log.Fatalf("Timer %v is already started", name)
		}
	}

	t.started = time.Now()
	t.running = true
}

func (c *Clock) Elapsed(name string) time.Duration {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := c.get(name)
	return time.Since(t.started)
}

func (c *Clock) Stop(name string) time.Duration {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := c.get(name)
	elapsed := time.Since(t.started)
	t.total += elapsed
	t.running = false
	return elapsed
}

func (c *Clock) get(name string) *Timer {
	t, ok := c.timers[name]
	if !ok {
		log.Fatalf("No such timer %v", name)
	}

	if !t.running {
		log.Fatalf("Timer %v is not running", name)
	}
	return t
}

func (c Clock) String() string {
	var sep string
	var buf bytes.Buffer

	for _, name := range c.names {
		var running string
		var total time.Duration
		t := c.timers[name]
		if t.running {
			total = t.total + time.Since(t.started)
			running = "*"
		} else {
			total = t.total
		}
		fmt.Fprintf(&buf, "%s%s=%.3f%s", sep, name, total.Seconds(), running)
		if sep == "" {
			sep = ", "
		}
	}

	return buf.String()
}
