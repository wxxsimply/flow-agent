package config

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignKlingJWTStructure(t *testing.T) {
	token, err := signKlingJWT("test-ak", "test-sk")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (any, error) {
		if tok.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			t.Fatalf("alg %s", tok.Method.Alg())
		}
		return []byte("test-sk"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims type")
	}
	if claims["iss"] != "test-ak" {
		t.Fatalf("iss=%v", claims["iss"])
	}
	now := time.Now().Unix()
	exp, _ := claims["exp"].(float64)
	if exp < float64(now) {
		t.Fatalf("exp in past")
	}
}
