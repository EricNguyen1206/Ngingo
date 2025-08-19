package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"Ngingo/internal/balancer"
	"Ngingo/internal/limiter"
	"Ngingo/internal/middleware"
	"Ngingo/internal/proxy"
	"Ngingo/internal/static"
)

// Minimal config via flags/env
type Config struct {
	ListenAddr    string
	StaticDir     string
	StaticPrefix  string
	ProxyPrefix   string
	UpstreamStr   string
	RPS           float64
	Burst         int
	TrustedHeader string
}

func main() {
	cfg := loadConfig()
	log.Printf("NginGo starting on %s", cfg.ListenAddr)

	rr, err := balancer.NewRoundRobin(cfg.UpstreamStr)
	if err != nil {
		log.Fatalf("failed to parse upstreams: %v", err)
	}

	limStore := limiter.NewStore(cfg.RPS, cfg.Burst)
	go limStore.CleanupLoop()

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static
	if cfg.StaticDir != "" {
		prefix := normalizePrefix(cfg.StaticPrefix)
		handler := http.StripPrefix(prefix, static.BuildStaticHandler(cfg.StaticDir))
		mux.Handle(prefix, handler)
		log.Printf("Static: %s -> %s", prefix, path.Clean(cfg.StaticDir))
	}

	// Reverse proxy (if upstreams)
	if rr != nil && rr.Count() > 0 {
		pp := normalizePrefix(cfg.ProxyPrefix)
		handler := http.StripPrefix(pp, proxy.BuildProxyHandler(rr))
		mux.Handle(pp, handler)
		log.Printf("Proxy: %s -> %d upstream(s)", pp, rr.Count())
	}

	// Page root info
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML()))
	})

	// Compose middleware: logging -> rateLimit -> mux
	var handler http.Handler = mux
	handler = limStore.Middleware(cfg.TrustedHeader)(handler)
	handler = middleware.Logging(handler)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func loadConfig() Config {
	var cfg Config
	flag.StringVar(&cfg.ListenAddr, "listen", getEnv("NGINGO_LISTEN", ":8080"), "Address to listen on, e.g. :8080")
	flag.StringVar(&cfg.StaticDir, "staticDir", getEnv("NGINGO_STATIC_DIR", ""), "Directory to serve as static content (empty to disable)")
	flag.StringVar(&cfg.StaticPrefix, "staticPrefix", getEnv("NGINGO_STATIC_PREFIX", "/static"), "URL prefix for static files")
	flag.StringVar(&cfg.ProxyPrefix, "proxyPrefix", getEnv("NGINGO_PROXY_PREFIX", "/proxy"), "URL prefix to reverse proxy to upstreams")
	flag.StringVar(&cfg.UpstreamStr, "upstreams", getEnv("NGINGO_UPSTREAMS", ""), "Comma-separated upstream URLs")
	flag.Float64Var(&cfg.RPS, "rps", getEnvFloat("NGINGO_RPS", 5), "Rate limit: req/s per client IP")
	flag.IntVar(&cfg.Burst, "burst", getEnvInt("NGINGO_BURST", 10), "Rate limit: burst per client IP")
	flag.StringVar(&cfg.TrustedHeader, "trustedHeader", getEnv("NGINGO_TRUSTED_HEADER", ""), "Optional header for real client IP (e.g., X-Real-IP)")
	flag.Parse()
	return cfg
}

func indexHTML() string {
	return `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>NginGo</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial; margin: 2rem; }
    code { background: #f4f4f5; padding: 2px 6px; border-radius: 6px; }
    .card { border: 1px solid #e4e4e7; border-radius: 12px; padding: 1rem 1.25rem; margin-bottom: 1rem; }
    h1 { margin-top: 0; }
  </style>
</head>
<body>
  <h1>🧩 NginGo – Mini Nginx Clone</h1>
  <div class="card"><strong>Static:</strong> <code>/static/</code></div>
  <div class="card"><strong>Proxy:</strong> <code>/proxy/</code> (round‑robin)</div>
  <div class="card"><strong>Health:</strong> <code>/healthz</code></div>
  <div class="card"><strong>Rate Limit:</strong> per‑IP</div>
</body>
</html>`
}

// --- small helpers (local to main to avoid extra package config) ---
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var x int
		if _, err := fmt.Sscanf(v, "%d", &x); err == nil {
			return x
		}
	}
	return def
}
func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		var x float64
		if _, err := fmt.Sscanf(v, "%f", &x); err == nil {
			return x
		}
	}
	return def
}
func normalizePrefix(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}
