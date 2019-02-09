package main

import (
	"bytes"
	"fmt"
	"github.com/go-redis/redis"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type rateLimitTransport struct {
	limit int64
	interval time.Duration
	redisClient *redis.Client
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	count, _ := t.redisClient.Eval(`local v=redis.call("incrby",KEYS[1],1) if v==1 then redis.call("expire",KEYS[1],5) end return v`, []string{ip}).Result()
	if  count.(int64) > t.limit {
		return &http.Response{
			StatusCode: 503,
			Body: ioutil.NopCloser(bytes.NewBufferString("rate limit exceeded")),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func buildRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	client.FlushAll()
	return client
}

func main() {
	var transport rateLimitTransport
	transport.redisClient = buildRedisClient()
	transport.interval = 5 * time.Second
	transport.limit = 3
	var reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL, _ = url.Parse(os.Getenv("TARGET_URL"))
			req.Host = req.URL.Host
		},
		Transport: &transport,
	}
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		reverseProxy.ServeHTTP(w, r)
	})
	fmt.Println("listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
