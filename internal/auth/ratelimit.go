package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter 简单内存滑动窗口限流（单进程）。
type RateLimiter struct {
	mu     sync.Mutex
	events map[string][]time.Time
	max    int
	window time.Duration
}

// NewRateLimiter 创建限流器：window 内最多 max 次。
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max <= 0 {
		max = 10
	}
	if window <= 0 {
		window = time.Hour
	}
	return &RateLimiter{
		events: make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

// Allow 检查 key 是否未超限；通过则记录一次。
func (l *RateLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	prev := l.events[key]
	var kept []time.Time
	for _, t := range prev {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.events[key] = kept
		return false
	}
	kept = append(kept, now)
	l.events[key] = kept
	return true
}

// ClientIP 从请求解析客户端 IP（支持 X-Forwarded-For 首段）。
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

// ErrRateLimited 请求过于频繁。
var ErrRateLimited = fmt.Errorf("请求过于频繁，请稍后再试")
