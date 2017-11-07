package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	KB = 1 << 10
	MB = 1 << 20
)

func main() {
	sizeMB := flag.Int64(
		"size-mb",
		-1,
		`upload size in MiB. Must be specied when uploading character special
		file like /dev/zero. Use file size by default.`)
	blocksizeKB := flag.Int64(
		"blocksize-kb",
		4,
		`block size in KiB. Unused since net/http/transport.go use hard-coded
		value of 4 KiB. Defined to statisfy upload tool interface.`)
	flag.Parse()

	size := *sizeMB * MB

	if *blocksizeKB != 4 {
		fmt.Printf("Ignoring blocksize-kb (%d), not implemnted\n", *blocksizeKB)
	}

	if flag.NArg() != 2 {
		fmt.Printf("Usage: upload-go [options] filename url\n")
		os.Exit(2)
	}

	filename := flag.Arg(0)
	url := flag.Arg(1)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	if size < 0 {
		fi, err := file.Stat()
		if err != nil {
			log.Fatal(err)
		}
		size = fi.Size()
		if size == 0 {
			log.Fatalf("Cannot determine %q size, please specify --size-mb", filename)
		}
	}

	req, err := http.NewRequest(http.MethodPut, url, io.LimitReader(file, size))
	if err != nil {
		log.Fatal(err)
	}

	req.ContentLength = size

	// We don't care about certificates validation in this test
	// WriteBufSize requires https://go-review.googlesource.com/#/c/go/+/76410/
	insecureTransport := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
		WriteBufSize:       128 * 1024,
	}

	client := &http.Client{Transport: insecureTransport}

	start := time.Now()

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	elapsed := time.Since(start).Seconds()

	if res.StatusCode != 200 {
		log.Fatalf("Request failed: %s", res.Status)
	}

	fmt.Printf("Uploaded %.2f MiB in %.2f seconds (%.2f MiB/s)\n",
		float64(size)/float64(MB), elapsed, float64(size)/float64(MB)/elapsed)
}
