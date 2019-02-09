package main

import (
	"bytes"
	"fmt"
	"github.com/go-redis/redis"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
)

const (
	redisLuaIncrScript  = `local v=redis.call("incrby",KEYS[1],1) if v==1 then redis.call("expire",KEYS[1],KEYS[2]) end return v`
	rateLimitErrorMsg   = "rate limit exceeded"
	rateLimitStatusCode = 503
	runningMsg          = "reverse-proxy listening on port 8080"
)

var (
	backendUrl   = os.Getenv("BACKEND_URL")
	redisHost    = os.Getenv("REDIS_HOST")
	redisPort    = os.Getenv("REDIS_PORT")
	intervalSecs = os.Getenv("INTERVAL_SECS")
	limit        = os.Getenv("LIMIT")
)

type rateLimitTransport struct {
	limit int
	interval int
	redisClient *redis.Client
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	count, _ := t.redisClient.Eval(redisLuaIncrScript, []string{ip, strconv.Itoa(t.limit)}).Result()
	if  count.(int64) > int64(t.limit) {
		return &http.Response{
			StatusCode: rateLimitStatusCode,
			Body: ioutil.NopCloser(bytes.NewBufferString(rateLimitErrorMsg)),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func main() {
	var transport rateLimitTransport
	transport.redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	transport.interval, _ = strconv.Atoi(intervalSecs)
	transport.limit, _ = strconv.Atoi(limit)
	var reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL, _ = url.Parse(backendUrl)
			req.Host = req.URL.Host
		},
		Transport: &transport,
	}
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		reverseProxy.ServeHTTP(w, r)
	})
	fmt.Println(runningMsg)
	http.ListenAndServe(":8080", nil)
}
