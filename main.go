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
	redisLuaIncrScript = `local v=redis.call("incrby",KEYS[1],1) if v==1 then redis.call("expire",KEYS[1],KEYS[2]) end return v`
	rateLimitErrorMsg  = "rate limit exceeded"
	runningMsg = "listening on port 8080"
)

var (
	redisHost = os.Getenv("REDIS_HOST")
	redisPort = os.Getenv("REDIS_POST")
	targetUrl = os.Getenv("TARGET_URL")
	intervalSecs = os.Getenv("INTERVAL_SECS")
	limit = os.Getenv("LIMIT")
)

type rateLimitTransport struct {
	limit int64
	interval int
	redisClient *redis.Client
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	count, _ := t.redisClient.Eval(redisLuaIncrScript, []string{ip, strconv.Itoa(t.interval)}).Result()
	if  count.(int64) > t.limit {
		return &http.Response{
			StatusCode: 503,
			Body: ioutil.NopCloser(bytes.NewBufferString(rateLimitErrorMsg)),
		}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

func buildRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       0,
	})
	client.FlushAll()
	return client
}

func main() {
	var err error
	var transport rateLimitTransport
	transport.redisClient = buildRedisClient()
	transport.interval, err = strconv.Atoi(intervalSecs)
	if err != nil {
		panic(err)
	}
	transport.limit, err = strconv.ParseInt(limit, 10, 64)
	if err != nil {
		panic(err)
	}
	var reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL, _ = url.Parse(targetUrl)
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
