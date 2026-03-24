package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// PerPlayerRateLimit returns a middleware that enforces a per-player token-bucket
// rate limit. playerID is extracted from the request context via keyFn.
// r is the sustained rate (requests/sec); burst is the allowed burst size.
// Stale limiters are pruned every cleanupInterval.
func PerPlayerRateLimit(r rate.Limit, burst int, cleanupInterval time.Duration, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	type entry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var mu sync.Mutex
	limiters := make(map[string]*entry)

	// Background cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-cleanupInterval)
			mu.Lock()
			for k, e := range limiters {
				if e.lastSeen.Before(cutoff) {
					delete(limiters, k)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		e, ok := limiters[key]
		if !ok {
			e = &entry{limiter: rate.NewLimiter(r, burst)}
			limiters[key] = e
		}
		e.lastSeen = time.Now()
		return e.limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			key := keyFn(req)
			if key == "" {
				next.ServeHTTP(w, req)
				return
			}
			if !getLimiter(key).Allow() {
				http.Error(w, `{"code":"rate_limited","message":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
