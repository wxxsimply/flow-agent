package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// SMSClient 短信发送接口。
type SMSClient interface {
	SendOTP(ctx context.Context, phone, code string) error
}

// NewSMSClient 按配置创建短信客户端。
func NewSMSClient(cfg SMSConfig) SMSClient {
	switch cfg.Provider {
	case "aliyun":
		return &AliyunSMS{cfg: cfg}
	default:
		return &DevSMS{code: cfg.DevCode}
	}
}

// DevSMS 开发模式：不真正发短信，验证码固定或可配置。
type DevSMS struct {
	code string
}

func (d *DevSMS) SendOTP(ctx context.Context, phone, code string) error {
	_ = ctx
	slog.Info("dev sms otp", "phone", maskPhone(phone), "code", code)
	return nil
}

// AliyunSMS 阿里云短信（需配置 AccessKey 与模板）。
type AliyunSMS struct {
	cfg SMSConfig
}

func (a *AliyunSMS) SendOTP(ctx context.Context, phone, code string) error {
	if a.cfg.AccessKeyID == "" || a.cfg.AccessKeySecret == "" {
		return fmt.Errorf("aliyun sms: missing access key")
	}
	if a.cfg.SignName == "" || a.cfg.TemplateCode == "" {
		return fmt.Errorf("aliyun sms: missing sign name or template code")
	}
	slog.Warn("aliyun sms not fully wired; set FLOWAGENT_SMS_PROVIDER=dev for testing",
		"phone", maskPhone(phone), "template", a.cfg.TemplateCode)
	return fmt.Errorf("aliyun sms: configure SDK or use FLOWAGENT_SMS_PROVIDER=dev")
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// AuthService 短信登录业务。
type AuthService struct {
	store *Store
	sms   SMSClient
	cfg   Config
}

// NewAuthService 创建认证服务。
func NewAuthService(store *Store, sms SMSClient, cfg Config) *AuthService {
	return &AuthService{store: store, sms: sms, cfg: cfg}
}

// SendSMSCode 发送登录验证码。
func (a *AuthService) SendSMSCode(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if phone == "" {
		return fmt.Errorf("invalid phone number")
	}
	last, err := a.store.LastOTPSentAt(phone)
	if err != nil {
		return err
	}
	if !last.IsZero() && time.Since(last) < a.cfg.OTP.ResendGap {
		return fmt.Errorf("请稍后再试")
	}
	code := a.cfg.SMS.DevCode
	if a.cfg.SMS.Provider != "dev" {
		var genErr error
		code, genErr = generateOTP(a.cfg.OTP.CodeLen)
		if genErr != nil {
			return genErr
		}
	}
	expires := time.Now().UTC().Add(a.cfg.OTP.TTL)
	if err := a.store.SaveOTP(phone, code, expires); err != nil {
		return err
	}
	return a.sms.SendOTP(ctx, phone, code)
}

// LoginWithSMS 验证码登录，返回 JWT。
func (a *AuthService) LoginWithSMS(ctx context.Context, phone, code string) (string, *User, error) {
	_ = ctx
	phone = normalizePhone(phone)
	if phone == "" {
		return "", nil, fmt.Errorf("invalid phone number")
	}
	if !(a.cfg.SMS.Provider == "dev" && code == a.cfg.SMS.DevCode) {
		ok, err := a.store.VerifyOTP(phone, code)
		if err != nil {
			return "", nil, err
		}
		if !ok {
			return "", nil, fmt.Errorf("验证码错误或已过期")
		}
	}
	user, err := a.store.FindOrCreateUser(phone)
	if err != nil {
		return "", nil, err
	}
	token, err := IssueToken(a.cfg.JWTSecret, user.ID, user.Phone, 7*24*time.Hour)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}
