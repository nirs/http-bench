package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	MB      = 1 << 20
	GB      = 1 << 30
	bufSize = 1 * MB
)

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServeTLS(":8000", "cert.pem", "key.pem", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	logEvent(r, "START", "(%.2fg)", float64(r.ContentLength)/float64(GB))

	if r.Method != "PUT" {
		fail(w, r, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == -1 {
		fail(w, r, "Content-Length required", http.StatusBadRequest)
		return
	}

	start := time.Now()

	if _, err := discard(r); err != nil {
		fail(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	elapsed := time.Since(start).Seconds()

	logEvent(r, "FINISH", "(%.2fg in %.2f seconds, %.2fm/s)",
		float64(r.ContentLength)/float64(GB),
		elapsed,
		float64(r.ContentLength)/float64(MB)/elapsed)
}

func fail(w http.ResponseWriter, r *http.Request, msg string, code int) {
	logEvent(r, "ERROR", msg)
	http.Error(w, msg, code)
}

func logEvent(r *http.Request, event string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] %s %s %q: %s\n", r.RemoteAddr, event, r.Method, r.URL.Path, message)
}

func discard(r *http.Request) (n int64, err error) {
	buf := make([]byte, bufSize)
	reader := io.LimitReader(r.Body, r.ContentLength)
	return io.CopyBuffer(ioutil.Discard, reader, buf)
}
