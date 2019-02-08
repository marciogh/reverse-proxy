package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type rateLimitTransport struct {
	limit      int
	countMutex sync.RWMutex
	countMap   map[string]int
}

func (r *rateLimitTransport) count(remoteAddr string) int {
	r.countMutex.Lock()
	defer r.countMutex.Unlock()
	r.countMap[remoteAddr]++
	return r.countMap[remoteAddr]
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	if t.count(ip) > t.limit {
		return &http.Response{
			StatusCode: 503,
			Body: ioutil.NopCloser(bytes.NewBufferString("Rate limit exceeded")),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func main() {
	var transport rateLimitTransport
	var reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL, _ = url.Parse(os.Getenv("TARGET_URL"))
			req.Host = req.URL.Host
		},
		Transport: &transport,
	}
	transport.limit = 3
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		reverseProxy.ServeHTTP(w, r)
	})
	go func() {
		for {
			transport.countMap = make(map[string]int)
			time.Sleep(5 * time.Second)
		}
	}()
	fmt.Println("Reverse Proxy listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
