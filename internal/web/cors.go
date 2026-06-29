package web

import (
	"net/http"
	"os"
	"strings"
)

// corsConfig holds allowed Origin patterns for CORS responses.
type corsConfig struct {
	exact  map[string]struct{}
	prefix []string // e.g. http://127.0.0.1:
}

func loadCORSConfig(desktopMode bool) corsConfig {
	raw := strings.TrimSpace(os.Getenv("FLOWAGENT_CORS_ORIGINS"))
	if raw == "" {
		if desktopMode {
			return corsConfig{
				prefix: []string{
					"http://127.0.0.1:",
					"http://localhost:",
				},
			}
		}
		return corsConfig{
			exact: map[string]struct{}{
				"http://localhost:8080":  {},
				"http://127.0.0.1:8080": {},
			},
			prefix: []string{
				"http://127.0.0.1:",
				"http://localhost:",
			},
		}
	}
	if raw == "*" {
		if strings.EqualFold(os.Getenv("FLOWAGENT_CORS_ALLOW_ALL"), "true") {
			return corsConfig{exact: map[string]struct{}{"*": {}}}
		}
		return corsConfig{}
	}
	cfg := corsConfig{exact: map[string]struct{}{}}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasSuffix(part, "*") {
			cfg.prefix = append(cfg.prefix, strings.TrimSuffix(part, "*"))
		} else {
			cfg.exact[part] = struct{}{}
		}
	}
	return cfg
}

func (c corsConfig) allowOrigin(origin string) string {
	if origin == "" {
		return ""
	}
	if _, ok := c.exact["*"]; ok {
		return "*"
	}
	if _, ok := c.exact[origin]; ok {
		return origin
	}
	for _, p := range c.prefix {
		if strings.HasPrefix(origin, p) {
			return origin
		}
	}
	return ""
}

func corsMiddleware(next http.Handler, desktopMode bool) http.Handler {
	cfg := loadCORSConfig(desktopMode)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed := cfg.allowOrigin(origin); allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			if allowed != "*" {
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
