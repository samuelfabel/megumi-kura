package captcha

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIssueAndVerifyMath(t *testing.T) {
	s := New(Options{Provider: ProviderMath, SessionHMACSecret: []byte("test-secret"), TTL: time.Minute})
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if ch.Provider != ProviderMath || ch.Token == "" || ch.Question == "" {
		t.Fatal("empty challenge")
	}
	var a, b int
	if _, err := fmt.Sscanf(ch.Question, "%d + %d", &a, &b); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(context.Background(), ch.Token, fmt.Sprintf("%d", a+b), ""); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := s.Verify(context.Background(), ch.Token, fmt.Sprintf("%d", a+b), ""); err != ErrUsed {
		t.Fatalf("expected ErrUsed, got %v", err)
	}
}

func TestRejectWrongAnswer(t *testing.T) {
	s := New(Options{Provider: ProviderMath, SessionHMACSecret: []byte("test-secret"), TTL: time.Minute})
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(context.Background(), ch.Token, "99999", ""); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestExpired(t *testing.T) {
	s := New(Options{Provider: ProviderMath, SessionHMACSecret: []byte("test-secret"), TTL: 20 * time.Millisecond})
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	var a, b int
	_, _ = fmt.Sscanf(ch.Question, "%d + %d", &a, &b)
	if err := s.Verify(context.Background(), ch.Token, fmt.Sprintf("%d", a+b), ""); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestGoogleIssueNeedsKeys(t *testing.T) {
	s := New(Options{Provider: ProviderGoogle})
	if _, err := s.Issue(); err != ErrMisconfigured {
		t.Fatalf("expected ErrMisconfigured, got %v", err)
	}
	s = New(Options{Provider: ProviderGoogle, SiteKey: "site", SecretKey: "secret"})
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if ch.Provider != ProviderGoogle || ch.SiteKey != "site" || ch.Token != "" {
		t.Fatalf("unexpected challenge: %+v", ch)
	}
}

func TestSiteVerifySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("secret") != "sekrit" || r.Form.Get("response") != "tok" {
			t.Fatalf("bad form: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	s := New(Options{Provider: ProviderCloudflare, SiteKey: "site", SecretKey: "sekrit"})
	s.HTTP = srv.Client()
	// monkey-patch by verifying against custom endpoint via temporary method path:
	// call verifySiteVerify directly
	if err := s.verifySiteVerify(context.Background(), srv.URL, "tok", "1.2.3.4"); err != nil {
		t.Fatal(err)
	}
}

func TestSiteVerifyFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer srv.Close()
	s := New(Options{Provider: ProviderGoogle, SiteKey: "site", SecretKey: "sekrit"})
	s.HTTP = srv.Client()
	if err := s.verifySiteVerify(context.Background(), srv.URL, "bad", ""); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestDefaultProviderMath(t *testing.T) {
	s := New(Options{SessionHMACSecret: []byte("x")})
	if s.Provider != ProviderMath {
		t.Fatalf("got %s", s.Provider)
	}
	if !strings.EqualFold(s.PublicProvider(), ProviderMath) {
		t.Fatal("public provider")
	}
}
