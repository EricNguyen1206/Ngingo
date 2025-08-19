package util

import (
	"net"
	"net/http"
	"strings"
)

// RealClientIP: lấy IP thực (ưu tiên trusted header nếu có)
func RealClientIP(r *http.Request, trustedHeader string) string {
	if trustedHeader != "" {
		if ip := strings.TrimSpace(r.Header.Get(trustedHeader)); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func SchemeOf(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}

func ClientIPFrom(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
