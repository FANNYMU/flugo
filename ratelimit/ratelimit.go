package ratelimit

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"flugo.com/response"
	"flugo.com/router"
)

type Limiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	max      int
	window   time.Duration
}

type Config struct {
	Requests int
	Window   time.Duration
	KeyFunc  func(*http.Request) string
}

var DefaultLimiter *Limiter

func Init(max int, window time.Duration) {
	DefaultLimiter = NewLimiter(max, window)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			DefaultLimiter.cleanup()
		}
	}()
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{
		requests: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	requests := l.requests[key]

	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	if len(validRequests) >= l.max {
		l.requests[key] = validRequests
		return false
	}

	validRequests = append(validRequests, now)
	l.requests[key] = validRequests

	return true
}

func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	for key, requests := range l.requests {
		validRequests := make([]time.Time, 0)
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(l.requests, key)
		} else {
			l.requests[key] = validRequests
		}
	}
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.requests, key)
}

func (l *Limiter) Remaining(key string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	requests := l.requests[key]
	if requests == nil {
		return l.max
	}

	now := time.Now()
	cutoff := now.Add(-l.window)

	validCount := 0
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validCount++
		}
	}

	remaining := l.max - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func Limit(requests int, window time.Duration) router.MiddlewareFunc {
	return LimitWithConfig(Config{
		Requests: requests,
		Window:   window,
		KeyFunc:  getClientIP,
	})
}

func LimitByUser(requests int, window time.Duration) router.MiddlewareFunc {
	return LimitWithConfig(Config{
		Requests: requests,
		Window:   window,
		KeyFunc: func(r *http.Request) string {
			userID := r.Header.Get("X-Current-User")
			if userID == "" {
				return getClientIP(r)
			}
			return "user:" + userID
		},
	})
}

func LimitByEndpoint(requests int, window time.Duration) router.MiddlewareFunc {
	return LimitWithConfig(Config{
		Requests: requests,
		Window:   window,
		KeyFunc: func(r *http.Request) string {
			return fmt.Sprintf("%s:%s:%s", getClientIP(r), r.Method, r.URL.Path)
		},
	})
}

func LimitWithConfig(config Config) router.MiddlewareFunc {
	limiter := NewLimiter(config.Requests, config.Window)

	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := config.KeyFunc(r)

			if !limiter.Allow(key) {
				remaining := limiter.Remaining(key)
				resetTime := time.Now().Add(config.Window).Unix()

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))

				response.TooManyRequests(w, "Rate limit exceeded")
				return
			}

			remaining := limiter.Remaining(key)
			resetTime := time.Now().Add(config.Window).Unix()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

			next(w, r)
		}
	}
}

func GlobalLimit(requests int, window time.Duration) router.MiddlewareFunc {
	return Limit(requests, window)
}

func Allow(key string) bool {
	if DefaultLimiter == nil {
		return true
	}
	return DefaultLimiter.Allow(key)
}

func Reset(key string) {
	if DefaultLimiter != nil {
		DefaultLimiter.Reset(key)
	}
}

func Remaining(key string) int {
	if DefaultLimiter == nil {
		return 100
	}
	return DefaultLimiter.Remaining(key)
}
