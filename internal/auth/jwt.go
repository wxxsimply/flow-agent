package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userContextKey contextKey = "authUser"

// Claims JWT 载荷。
type Claims struct {
	UserID string `json:"uid"`
	Phone  string `json:"phone"`
	jwt.RegisteredClaims
}

// IssueToken 签发 JWT。
func IssueToken(secret, userID, phone string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Phone:  phone,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseToken 解析并校验 JWT。
func ParseToken(secret, token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// UserFromRequest 从 Authorization Bearer 或 cookie 读取用户 claims。
func UserFromRequest(r *http.Request, secret string) (*Claims, error) {
	token := bearerToken(r)
	if token == "" {
		return nil, fmt.Errorf("missing token")
	}
	return ParseToken(secret, token)
}

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if c, err := r.Cookie("flowagent_token"); err == nil {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

// Middleware 校验 JWT；auth 未启用时直接放行。
func Middleware(cfg Config, next http.Handler) http.Handler {
	if !cfg.Enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := UserFromRequest(r, cfg.JWTSecret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := contextWithUser(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext 从请求 context 读取 claims。
func UserFromContext(r *http.Request) (*Claims, bool) {
	v := r.Context().Value(userContextKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Claims)
	return c, ok
}

func contextWithUser(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// RequireUser 在 auth 启用时必须已登录。
func RequireUser(r *http.Request, cfg Config) (*Claims, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if c, ok := UserFromContext(r); ok {
		return c, nil
	}
	return nil, fmt.Errorf("unauthorized")
}
