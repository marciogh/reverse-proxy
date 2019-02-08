package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type rateLimitTransport struct {
	limit int32
	count int32
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.count++
	if t.count > t.limit {
		return &http.Response{
			StatusCode: 503,
			Body: ioutil.NopCloser(bytes.NewBufferString("Rate limit exceeded")),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

var transport rateLimitTransport

var reverseProxy = &httputil.ReverseProxy{
	Director: func(req *http.Request) {
		req.URL, _ = url.Parse(os.Getenv("TARGET_URL"))
		req.Host = req.URL.Host
	},
	Transport: &transport,
}

func main() {
	transport.limit = 5
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		reverseProxy.ServeHTTP(w, r)
	})
	fmt.Println("Reverse Proxy listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
