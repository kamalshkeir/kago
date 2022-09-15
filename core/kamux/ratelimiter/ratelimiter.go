package ratelimiter

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var banned = sync.Map{}
var LIMITER_TOKENS = 50
var LIMITER_TIMEOUT = 5 * time.Minute
var LIMITER = func(next http.Handler) http.Handler {
	var limiter = rate.NewLimiter(1, LIMITER_TOKENS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := banned.Load(r.RemoteAddr)
		if ok {
			if time.Since(v.(time.Time)) > LIMITER_TIMEOUT {
				banned.Delete(r.RemoteAddr)
			} else {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 5 MINUTES </h1>"))
				banned.Store(r.RemoteAddr, time.Now())
				return
			}
		}
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("<h1>YOU DID TOO MANY REQUEST, YOU HAVE BEEN BANNED FOR 5 MINUTES </h1>"))
			banned.Store(r.RemoteAddr, time.Now())
			return
		}
		next.ServeHTTP(w, r)
	})
}

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	return i
}

// AddIP creates a new rate limiter and adds it to the ips map,
// using the IP address as the key
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.ips[ip] = limiter

	return limiter
}

// GetLimiter returns the rate limiter for the provided IP address if it exists.
// Otherwise calls AddIP to add IP address to the map
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	i.mu.Unlock()

	return limiter
}
