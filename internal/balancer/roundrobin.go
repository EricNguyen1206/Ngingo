package balancer

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type RoundRobin struct {
	backends []*url.URL
	mu       sync.Mutex
	idx      int
}

func NewRoundRobin(raw string) (*RoundRobin, error) {
	if strings.TrimSpace(raw) == "" {
		return &RoundRobin{}, nil
	}
	parts := strings.Split(raw, ",")
	var urls []*url.URL
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		u, err := url.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream '%s': %w", p, err)
		}
		urls = append(urls, u)
	}
	return &RoundRobin{backends: urls}, nil
}

func (rr *RoundRobin) Next() *url.URL {
	if rr == nil || len(rr.backends) == 0 {
		return nil
	}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	u := rr.backends[rr.idx%len(rr.backends)]
	rr.idx++
	return u
}

func (rr *RoundRobin) Count() int { return len(rr.backends) }
