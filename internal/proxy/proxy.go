package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"

	"Ngingo/internal/balancer"
	"Ngingo/internal/util"
)

// BuildProxyHandler: reverse proxy với round‑robin từ balancer
func BuildProxyHandler(rr *balancer.RoundRobin) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := rr.Next()
		if target == nil {
			http.Error(w, "no upstream configured", http.StatusBadGateway)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		origDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			origDirector(req)
			req.Host = target.Host
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Header.Set("X-Forwarded-Proto", util.SchemeOf(r))
			req.Header.Set("X-Real-IP", util.ClientIPFrom(r))
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			http.Error(w, "upstream error", http.StatusBadGateway)
		}
		proxy.ServeHTTP(w, r)
	})
}
