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
	fmt.Printf("[%s] START %s %q (%.2fg)\n",
		r.RemoteAddr, r.Method, r.URL.Path, float64(r.ContentLength)/float64(GB))

	if r.Method != "PUT" {
		fmt.Printf("[%s] ERROR %s %q: %s\n",
			r.Method, r.RemoteAddr, r.URL.Path, "Unsupported method")
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == -1 {
		fmt.Printf("[%s] ERROR %s %q: %s\n",
			r.Method, r.RemoteAddr, r.URL.Path, "Content-Length required")
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}
	start := time.Now()

	if _, err := drop(r); err != nil {
		fmt.Printf("[%s] ERROR PUT %q: %s\n",
			r.RemoteAddr,
			r.URL.Path,
			err.Error())
		http.Error(w, "Error receiving complete body", http.StatusBadRequest)
		return
	}

	elapsed := time.Since(start).Seconds()

	fmt.Printf("[%s] FINISH PUT %q (%.2fg in %.2f seconds, %.2fm/s)\n",
		r.RemoteAddr,
		r.URL.Path,
		float64(r.ContentLength)/float64(GB),
		elapsed,
		float64(r.ContentLength)/float64(MB)/elapsed)
}

func drop(r *http.Request) (n int64, err error) {
	buf := make([]byte, bufSize)
	reader := io.LimitReader(r.Body, r.ContentLength)

	return io.CopyBuffer(ioutil.Discard, reader, buf)
}
