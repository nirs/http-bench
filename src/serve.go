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

	"cio"
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
)

func main() {
	flag.Parse()

	if *debug {
		fmt.Printf("Using blocksizeKB=%v\n", *blocksizeKB)
		fmt.Printf("Using limitread=%v\n", *limitread)
		fmt.Printf("Using limitwrite=%v\n", *limitwrite)
		fmt.Printf("Using direct=%v\n", *direct)
		fmt.Printf("Using output=%v\n", *output)
		fmt.Printf("Using stats=%v\n", *stats)
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServeTLS(":8000", "cert.pem", "key.pem", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	logEvent(r, "START", "")

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
		logEvent(r, "INFO", "%v", clock)
	}

	logEvent(r, "FINISH", "%.2f MiB in %.2f seconds, %.2f MiB/s",
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
	log.Printf("[%s] %-7s %s %q %s", r.RemoteAddr, event, r.Method, r.URL.Path, message)
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
	// Should be safe to ignore error if Sync() succeeded.
	defer file.Close()

	src := &Reader{
		r:       r.Body,
		clock:   clock,
		measure: *stats || *debug || (*limitread != 0),
		limit:   *limitread,
	}

	dst := &Writer{
		w:       file,
		clock:   clock,
		measure: *stats || *debug || (*limitwrite != 0),
		limit:   *limitwrite,
	}

	clock.Start("copy")
	if n, err = cio.Copy(dst, src, *blocksizeKB*KB); err != nil {
		return n, err
	}
	clock.Stop("copy")

	if n != r.ContentLength {
		return n, fmt.Errorf("Incomplete write, copied %v bytes, expected %v bytes",
			n, r.ContentLength)
	}

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
		log.Printf("Synced in %.6f seconds", elapsed.Seconds())
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

type Reader struct {
	r       io.Reader
	clock   *Clock
	measure bool
	limit   int
}

func (r *Reader) Read(buf []byte) (n int, err error) {
	if r.measure {
		r.clock.Start("read")
	}
	n, err = io.ReadFull(r.r, buf)
	if r.measure {
		if r.limit > 0 {
			limitRate(n, r.clock.Elapsed("read"), r.limit)
		}
		elapsed := r.clock.Stop("read")
		if *debug {
			log.Printf("Read %d bytes in %.6f seconds", n, elapsed.Seconds())
		}
	}
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return
}

type Writer struct {
	w       io.Writer
	clock   *Clock
	measure bool
	limit   int
}

func (w *Writer) Write(buf []byte) (n int, err error) {
	if w.measure {
		w.clock.Start("write")
	}
	n, err = w.w.Write(buf)
	if w.measure {
		if w.limit > 0 {
			limitRate(n, w.clock.Elapsed("write"), w.limit)
		}
		elapsed := w.clock.Stop("write")
		if *debug {
			log.Printf("Wrote %d bytes in %.6f seconds", n, elapsed.Seconds())
		}
	}
	return
}

// limitRate limit operation rate by sleeping until the expected time.
// TODO: sleep little less time, so time.Since(start) returns the expected value.
func limitRate(n int, elapsed time.Duration, rate int) {
	expected := time.Duration(float64(n) / float64(MB) / float64(rate) * 1e09)
	if expected > elapsed {
		time.Sleep(expected - elapsed)
	}
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

func (c *Clock) String() string {
	var sep string
	var buf bytes.Buffer

	c.mutex.Lock()
	defer c.mutex.Unlock()

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
