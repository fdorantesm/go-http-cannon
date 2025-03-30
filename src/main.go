package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var requestCount int64

func main() {
	var (
		concurrency   = flag.Int("c", 1, "Number of parallel requests")
		numCPUs       = flag.Int("x", 1, "Multiplier for number of CPUs used for concurrency")
		timeLimit     = flag.Int("t", 0, "Test duration in seconds")
		requests      = flag.Int("n", 0, "Total number of requests")
		waitTime      = flag.Int("w", 0, "Waiting time between requests in ms")
		method        = flag.String("X", "GET", "HTTP method")
		headers       = flag.String("H", "", "HTTP headers separated by ';'")
		body          = flag.String("d", "", "Data to send in the request body (or '@<file>' to load from file)")
		timeout       = flag.Duration("timeout", 1*time.Second, "Timeout for HTTP requests")
		insecure      = flag.Bool("k", false, "Ignore SSL certificate validation")
		useMultipart  = flag.Bool("multipart", false, "Send request as multipart/form-data")
		multipartFile = flag.String("F", "", "File to upload as multipart/form-data (field 'file')")
	)

	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage: cannon [options] <URL>")
		flag.PrintDefaults()
		return
	}

	url := flag.Arg(0)

	var requestBody string
	if strings.HasPrefix(*body, "@") {
		fileName := (*body)[1:]
		content, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatalf("Error reading file %s: %v\n", fileName, err)
		}
		requestBody = string(content)
	} else {
		requestBody = *body
	}

	var total int64
	var errorCount int64
	client := &http.Client{
		Timeout: *timeout,
	}
	if *insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var once sync.Once

	if *timeLimit > 0 {
		go func() {
			time.Sleep(time.Duration(*timeLimit) * time.Second)
			once.Do(func() { close(stop) })
		}()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-interrupt
		once.Do(func() { close(stop) })
	}()

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				var req *http.Request
				var err error

				if *useMultipart && *multipartFile != "" {
					var buf bytes.Buffer
					writer := multipart.NewWriter(&buf)

					file, err := os.Open(*multipartFile)
					if err != nil {
						log.Printf("Error opening file: %v\n", err)
						return
					}
					defer file.Close()

					part, err := writer.CreateFormFile("file", filepath.Base(*multipartFile))
					if err != nil {
						log.Printf("Error creating form file: %v\n", err)
						return
					}
					io.Copy(part, file)
					writer.Close()

					req, err = http.NewRequest(*method, url, &buf)
					req.Header.Set("Content-Type", writer.FormDataContentType())
				} else {
					req, err = http.NewRequest(*method, url, strings.NewReader(requestBody))
				}

				if err == nil && *headers != "" {
					for _, h := range strings.Split(*headers, ";") {
						parts := strings.SplitN(h, ":", 2)
						if len(parts) == 2 {
							req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
						}
					}
				}

				resp, err := client.Do(req)
				if err == nil {
					count := atomic.AddInt64(&requestCount, 1)
					log.Printf("%6d: %s %s %d\n", count, req.Method, url, resp.StatusCode)
					resp.Body.Close()
					atomic.AddInt64(&total, 1)
				} else {
					log.Printf("Error making request: %v\n", err)
					atomic.AddInt64(&errorCount, 1)
				}
				if *waitTime > 0 {
					time.Sleep(time.Duration(*waitTime) * time.Millisecond)
				}
				if *requests > 0 && atomic.LoadInt64(&total) >= int64(*requests) {
					once.Do(func() { close(stop) })
					return
				}
			}
		}
	}

	numWorkers := *concurrency * *numCPUs
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	wg.Wait()
	fmt.Printf("Total: %d, Success: %d, Errors: %d\n", total+errorCount, total, errorCount)
}
