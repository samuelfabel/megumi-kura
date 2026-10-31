package captcha

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ProviderMath       = "math"
	ProviderGoogle     = "google"
	ProviderCloudflare = "cloudflare"
)

var (
	ErrInvalid       = errors.New("invalid captcha")
	ErrExpired       = errors.New("captcha expired")
	ErrUsed          = errors.New("captcha already used")
	ErrMisconfigured = errors.New("captcha provider misconfigured")
)

// Service issues/verifies captchas for one configured provider.
type Service struct {
	Provider string
	SiteKey  string
	Secret   string // provider secret, or HMAC secret for math
	TTL      time.Duration
	HTTP     *http.Client

	mu   sync.Mutex
	used map[string]time.Time // math nonce → expiry
}

type PublicChallenge struct {
	Provider string `json:"provider"`
	SiteKey  string `json:"site_key,omitempty"`
	Token    string `json:"captcha_token,omitempty"`
	Question string `json:"question,omitempty"`
}

type Options struct {
	Provider          string
	SiteKey           string
	SecretKey         string
	SessionHMACSecret []byte // used when provider=math
	TTL               time.Duration
}

func New(opts Options) *Service {
	p := strings.ToLower(strings.TrimSpace(opts.Provider))
	if p == "" {
		p = ProviderMath
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	secret := strings.TrimSpace(opts.SecretKey)
	if p == ProviderMath && secret == "" && len(opts.SessionHMACSecret) > 0 {
		secret = string(opts.SessionHMACSecret)
	}
	return &Service{
		Provider: p,
		SiteKey:  strings.TrimSpace(opts.SiteKey),
		Secret:   secret,
		TTL:      ttl,
		HTTP:     &http.Client{Timeout: 8 * time.Second},
		used:     map[string]time.Time{},
	}
}

func (s *Service) PublicProvider() string {
	return s.Provider
}

func (s *Service) PublicSiteKey() string {
	if s.Provider == ProviderMath {
		return ""
	}
	return s.SiteKey
}

// Issue returns a client challenge. For google/cloudflare only provider+site_key are needed.
func (s *Service) Issue() (PublicChallenge, error) {
	switch s.Provider {
	case ProviderGoogle, ProviderCloudflare:
		if s.SiteKey == "" || s.Secret == "" {
			return PublicChallenge{}, ErrMisconfigured
		}
		return PublicChallenge{Provider: s.Provider, SiteKey: s.SiteKey}, nil
	case ProviderMath:
		return s.issueMath()
	default:
		return PublicChallenge{}, fmt.Errorf("%w: unknown provider %q", ErrMisconfigured, s.Provider)
	}
}

// Verify validates a client response. For remote providers, answer is ignored.
func (s *Service) Verify(ctx context.Context, token, answer, remoteIP string) error {
	switch s.Provider {
	case ProviderGoogle:
		return s.verifySiteVerify(ctx, "https://www.google.com/recaptcha/api/siteverify", token, remoteIP)
	case ProviderCloudflare:
		return s.verifySiteVerify(ctx, "https://challenges.cloudflare.com/turnstile/v0/siteverify", token, remoteIP)
	case ProviderMath:
		return s.verifyMath(token, answer)
	default:
		return ErrMisconfigured
	}
}

func (s *Service) issueMath() (PublicChallenge, error) {
	if s.Secret == "" {
		return PublicChallenge{}, ErrMisconfigured
	}
	a, err := randInt(1, 12)
	if err != nil {
		return PublicChallenge{}, err
	}
	b, err := randInt(1, 12)
	if err != nil {
		return PublicChallenge{}, err
	}
	nonce, err := randomNonce(12)
	if err != nil {
		return PublicChallenge{}, err
	}
	exp := time.Now().Add(s.TTL).UnixMilli()
	payload := fmt.Sprintf("%d|%d|%d|%s", exp, a, b, nonce)
	sig := sign([]byte(s.Secret), payload)
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
	return PublicChallenge{
		Provider: ProviderMath,
		Token:    token,
		Question: fmt.Sprintf("%d + %d", a, b),
	}, nil
}

func (s *Service) verifyMath(token, answer string) error {
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
	if !hmac.Equal([]byte(parts[1]), []byte(sign([]byte(s.Secret), payload))) {
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

func (s *Service) verifySiteVerify(ctx context.Context, endpoint, token, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalid
	}
	if s.Secret == "" {
		return ErrMisconfigured
	}
	form := url.Values{}
	form.Set("secret", s.Secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var parsed struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
		Score      float64  `json:"score"` // recaptcha v3 (ignored for v2/turnstile)
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ErrInvalid
	}
	if !parsed.Success {
		return ErrInvalid
	}
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
