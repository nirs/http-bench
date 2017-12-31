package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
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
	count = flag.Int(
		"count",
		-1,
		`number of blocks to copy (default entire file).`)
	directInput = flag.Bool(
		"direct-input",
		false,
		"use direct I/O for input file")
	directOutput = flag.Bool(
		"direct-output",
		false,
		"use direct I/O for output file")
	debug = flag.Bool(
		"debug",
		false,
		`enable debug logging.`)

	input  string
	output string
)

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatalf("Missing input or output file")
	}

	input = flag.Arg(0)
	output = flag.Arg(1)

	if *debug {
		fmt.Printf("Using blocksizeKB=%v\n", *blocksizeKB)
		fmt.Printf("Using count=%v\n", *count)
		fmt.Printf("Using direct-input=%v\n", *directInput)
		fmt.Printf("Using direct-output=%v\n", *directOutput)
		fmt.Printf("Using input=%v\n", input)
		fmt.Printf("Using output=%v\n", output)
	}

	clock := newClock()
	clock.Start("total")

	inputFlags := os.O_RDONLY
	if *directInput {
		inputFlags |= syscall.O_DIRECT
	}
	inputFile, err := os.OpenFile(input, inputFlags, 0)
	if err != nil {
		log.Fatalf("Cannot open input file: %s", err)
	}
	defer inputFile.Close()

	outputFlags := os.O_RDWR
	if *directOutput {
		outputFlags |= syscall.O_DIRECT
	}
	outputFile, err := os.OpenFile(output, outputFlags, 0)
	if err != nil {
		log.Fatalf("Cannot open output file: %s", err)
	}
	defer outputFile.Close()

	var src io.Reader
	if *count > 0 {
		src = &Reader{r: io.LimitReader(inputFile, int64(*count)*int64(*blocksizeKB)*KB), clock: clock}
	} else {
		src = &Reader{r: inputFile, clock: clock}
	}
	dst := &Writer{w: outputFile, clock: clock}

	clock.Start("copy")
	copied, err := cio.Copy(dst, src, *blocksizeKB*KB)
	if err != nil {
		log.Fatalf("Copy error: %s", err)
	}
	clock.Stop("copy")

	clock.Start("sync")
	if err = outputFile.Sync(); err != nil {
		if errno(err) == syscall.EINVAL {
			// Sync to /dev/null fails with EINVAL; ignore it
			err = nil
		} else {
			log.Fatalf("Error syncing output: %s", err)
		}
	}
	clock.Stop("sync")

	elapsed := clock.Stop("total").Seconds()

	log.Printf("Stats: %v", clock)

	log.Printf("Copied %.2f MiB in %.2f seconds, %.2f MiB/s",
		float64(copied)/float64(MB),
		elapsed,
		float64(copied)/float64(MB)/elapsed)
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
	r     io.Reader
	clock *Clock
}

func (r *Reader) Read(buf []byte) (n int, err error) {
	r.clock.Start("read")
	n, err = io.ReadFull(r.r, buf)
	r.clock.Stop("read")
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return
}

type Writer struct {
	w     io.Writer
	clock *Clock
}

func (w *Writer) Write(buf []byte) (n int, err error) {
	w.clock.Start("write")
	n, err = w.w.Write(buf)
	w.clock.Stop("write")
	return
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
