package cio

import (
	"io"
	"sync"
	"unsafe"
)

const bufcount = 2

// Copy copies from src to dst until either EOF is reached
// on src or an error occurs. It returns the number of bytes
// copied and the first error encountered while copying, if any.
//
// A successful Copy returns err == nil, not err == EOF.
// Because Copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
//
// This version is optimized for direct I/O - the buffers used during
// the copy are alinged to 512 bytes, and the data is read from the
// reader and written to the writer concurrently.
func Copy(dst io.Writer, src io.Reader, bufsize int) (written int64, err error) {
	pool := make(chan *buffer, bufcount)
	work := make(chan *buffer, bufcount)

	for i := 0; i < bufcount; i++ {
		pool <- newBuffer(bufsize, 512)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for b := range work {
			n, ew := dst.Write(b.buf[:b.len])
			if n > 0 {
				written += int64(n)
			}
			if ew != nil {
				err = ew
				break
			}
			pool <- b
		}
		close(pool)
		wg.Done()
	}()

	for b := range pool {
		n, er := src.Read(b.buf)
		if n > 0 {
			b.len = n
			work <- b
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	close(work)
	wg.Wait()

	return
}

type buffer struct {
	buf []byte
	len int
}

func newBuffer(size int, align int) *buffer {
	buf := make([]byte, size+align)
	offset := 0
	remainder := int(uintptr(unsafe.Pointer(&buf[0])) & uintptr(align-1))
	if remainder != 0 {
		offset = align - remainder
	}
	return &buffer{buf: buf[offset : offset+size]}
}
