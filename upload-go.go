package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	MB = 1 << 20
	GB = 1 << 30
)

func main() {
	sizeGB, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	size := sizeGB * GB
	url := os.Args[2]

	file, err := os.Open("/dev/zero")
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPut, url, io.LimitReader(file, size))
	if err != nil {
		log.Fatal(err)
	}

	req.ContentLength = size

	// We don't care about certificates validation in this test
	insecureTransport := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
	}

	client := &http.Client{Transport: insecureTransport}

	start := time.Now()

	// TODO: find a way to configure the copy buffer size:
	// strace show this write 4k chunks:
	// [pid 32264] read(3, "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"..., 4096) = 4096
	// [pid 32264] write(4, "\27\3\3\20\30\0\0\0\0\0\3l\"w\201\360W\307F=\215Zj&\6hj\253\343\20EN"..., 4125) = 4125

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	elapsed := time.Since(start).Seconds()

	if res.StatusCode != 200 {
		log.Fatalf("Request failed: %s", res.Status)
	}

	fmt.Printf("Uploaded %.2fg in %.2f seconds (%.2fm/s)\n",
		float64(size)/float64(GB), elapsed, float64(size)/float64(MB)/elapsed)
}
