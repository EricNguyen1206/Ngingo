package limiter

import (
	"net/http"
	"sync"
	"time"

	"Ngingo/internal/util"

	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Store struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rps      rate.Limit
	burst    int
	lifetime time.Duration
}

func NewStore(rps float64, burst int) *Store {
	return &Store{
		clients:  make(map[string]*clientLimiter),
		rps:      rate.Limit(rps),
		burst:    burst,
		lifetime: 5 * time.Minute,
	}
}

func (s *Store) getLimiter(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.clients[key]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}
	l := rate.NewLimiter(s.rps, s.burst)
	s.clients[key] = &clientLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

func (s *Store) CleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for k, v := range s.clients {
			if time.Since(v.lastSeen) > s.lifetime {
				delete(s.clients, k)
			}
		}
		s.mu.Unlock()
	}
}

// Middleware for IP-based rate limiting
func (s *Store) Middleware(trustedHeader string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := util.RealClientIP(r, trustedHeader)
			if !s.getLimiter(ip).Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
