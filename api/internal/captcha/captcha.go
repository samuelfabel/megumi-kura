package captcha

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalid = errors.New("invalid captcha")
	ErrExpired = errors.New("captcha expired")
	ErrUsed    = errors.New("captcha already used")
)

// Service issues and verifies short-lived arithmetic challenges signed with HMAC.
// No third-party provider is required.
type Service struct {
	Secret []byte
	TTL    time.Duration

	mu   sync.Mutex
	used map[string]time.Time // nonce → expiry
}

type Challenge struct {
	Token    string `json:"captcha_token"`
	Question string `json:"question"`
}

func New(secret []byte, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Service{
		Secret: secret,
		TTL:    ttl,
		used:   map[string]time.Time{},
	}
}

func (s *Service) Issue() (Challenge, error) {
	a, err := randInt(1, 12)
	if err != nil {
		return Challenge{}, err
	}
	b, err := randInt(1, 12)
	if err != nil {
		return Challenge{}, err
	}
	nonce, err := randomNonce(12)
	if err != nil {
		return Challenge{}, err
	}
	exp := time.Now().Add(s.TTL).UnixMilli()
	payload := fmt.Sprintf("%d|%d|%d|%s", exp, a, b, nonce)
	sig := sign(s.Secret, payload)
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
	return Challenge{
		Token:    token,
		Question: fmt.Sprintf("%d + %d", a, b),
	}, nil
}

func (s *Service) Verify(token, answer string) error {
	token = strings.TrimSpace(token)
	answer = strings.TrimSpace(answer)
	if token == "" || answer == "" {
		return ErrInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ErrInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalid
	}
	payload := string(raw)
	if !hmac.Equal([]byte(parts[1]), []byte(sign(s.Secret, payload))) {
		return ErrInvalid
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 4 {
		return ErrInvalid
	}
	exp, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return ErrInvalid
	}
	if time.Now().UnixMilli() > exp {
		return ErrExpired
	}
	a, err := strconv.Atoi(fields[1])
	if err != nil {
		return ErrInvalid
	}
	b, err := strconv.Atoi(fields[2])
	if err != nil {
		return ErrInvalid
	}
	nonce := fields[3]
	if nonce == "" {
		return ErrInvalid
	}

	got, err := strconv.Atoi(answer)
	if err != nil || got != a+b {
		return ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(time.Now())
	if _, ok := s.used[nonce]; ok {
		return ErrUsed
	}
	s.used[nonce] = time.UnixMilli(exp)
	return nil
}

func (s *Service) purgeLocked(now time.Time) {
	for n, exp := range s.used {
		if now.After(exp) {
			delete(s.used, n)
		}
	}
}

func sign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randInt(min, max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + min, nil
}

func randomNonce(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
