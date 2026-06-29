package auth

import (
	"testing"
	"time"
)

func TestNormalizePhone(t *testing.T) {
	if got := normalizePhone("138 0013 8000"); got != "13800138000" {
		t.Fatalf("got %q", got)
	}
	if normalizePhone("123") != "" {
		t.Fatal("expected invalid")
	}
}

func TestAuthStoreOTP(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	phone := "13800138000"
	code := "654321"
	expires := time.Now().UTC().Add(5 * time.Minute)
	if err := store.SaveOTP(phone, code, expires); err != nil {
		t.Fatal(err)
	}
	ok, err := store.VerifyOTP(phone, code)
	if err != nil || !ok {
		t.Fatalf("verify failed ok=%v err=%v", ok, err)
	}
	ok, _ = store.VerifyOTP(phone, code)
	if ok {
		t.Fatal("otp should be one-time")
	}
}

func TestAuthServiceDevLogin(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cfg := Config{
		Enabled:   true,
		JWTSecret: "test-secret",
		SMS: SMSConfig{
			Provider: "dev",
			DevCode:  "123456",
		},
		OTP: OTPConfig{TTL: 5 * time.Minute, ResendGap: time.Second, CodeLen: 6},
	}
	svc := NewAuthService(store, NewSMSClient(cfg.SMS), cfg)
	token, user, err := svc.LoginWithSMS(t.Context(), "13900139000", "123456")
	if err != nil || token == "" || user == nil {
		t.Fatalf("login failed: %v", err)
	}
	claims, err := ParseToken(cfg.JWTSecret, token)
	if err != nil || claims.UserID != user.ID {
		t.Fatalf("token parse failed: %v", err)
	}
}
