package auth

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// User 登录用户。
type User struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

// Store SQLite 用户与 OTP 存储。
type Store struct {
	db *sql.DB
}

// OpenStore 打开或创建 app.db。
func OpenStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "app.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	phone TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sms_otps (
	phone TEXT NOT NULL,
	code TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	used INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sms_otps_phone ON sms_otps(phone, created_at DESC);
`)
	return err
}

// FindOrCreateUser 按手机号查找或创建用户。
func (s *Store) FindOrCreateUser(phone string) (*User, error) {
	phone = normalizePhone(phone)
	if phone == "" {
		return nil, fmt.Errorf("invalid phone")
	}
	var u User
	var created string
	err := s.db.QueryRow(`SELECT id, phone, created_at FROM users WHERE phone = ?`, phone).
		Scan(&u.ID, &u.Phone, &created)
	if err == nil {
		u.CreatedAt, _ = time.Parse(time.RFC3339, created)
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	id := newUserID()
	now := time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO users(id, phone, created_at) VALUES(?,?,?)`, id, phone, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Phone: phone, CreatedAt: now}, nil
}

// GetUser 按 ID 查找用户。
func (s *Store) GetUser(id string) (*User, error) {
	var u User
	var created string
	err := s.db.QueryRow(`SELECT id, phone, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Phone, &created)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

// SaveOTP 保存验证码。
func (s *Store) SaveOTP(phone, code string, expires time.Time) error {
	phone = normalizePhone(phone)
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`INSERT INTO sms_otps(phone, code, expires_at, created_at, used) VALUES(?,?,?,?,0)`,
		phone, code, expires.UTC().Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return err
}

// LastOTPSentAt 最近一次发送时间。
func (s *Store) LastOTPSentAt(phone string) (time.Time, error) {
	phone = normalizePhone(phone)
	var created string
	err := s.db.QueryRow(
		`SELECT created_at FROM sms_otps WHERE phone = ? ORDER BY created_at DESC LIMIT 1`, phone,
	).Scan(&created)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, created)
}

// VerifyOTP 校验验证码（一次性）。
func (s *Store) VerifyOTP(phone, code string) (bool, error) {
	phone = normalizePhone(phone)
	code = strings.TrimSpace(code)
	var id int64
	var stored, expiresStr string
	var used int
	err := s.db.QueryRow(`
SELECT rowid, code, expires_at, used FROM sms_otps
WHERE phone = ? ORDER BY created_at DESC LIMIT 1`, phone).Scan(&id, &stored, &expiresStr, &used)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if used != 0 {
		return false, nil
	}
	expires, _ := time.Parse(time.RFC3339, expiresStr)
	if time.Now().UTC().After(expires) {
		return false, nil
	}
	if stored != code {
		return false, nil
	}
	_, err = s.db.Exec(`UPDATE sms_otps SET used = 1 WHERE rowid = ?`, id)
	return err == nil, err
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	p := b.String()
	if len(p) == 11 && strings.HasPrefix(p, "1") {
		return p
	}
	return ""
}

func newUserID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func generateOTP(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	var s strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		s.WriteByte(byte('0' + n.Int64()))
	}
	return s.String(), nil
}
