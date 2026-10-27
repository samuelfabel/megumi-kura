package captcha

import (
	"fmt"
	"testing"
	"time"
)

func TestIssueAndVerify(t *testing.T) {
	s := New([]byte("test-secret"), time.Minute)
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if ch.Token == "" || ch.Question == "" {
		t.Fatal("empty challenge")
	}
	var a, b int
	if _, err := fmt.Sscanf(ch.Question, "%d + %d", &a, &b); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(ch.Token, fmt.Sprintf("%d", a+b)); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := s.Verify(ch.Token, fmt.Sprintf("%d", a+b)); err != ErrUsed {
		t.Fatalf("expected ErrUsed, got %v", err)
	}
}

func TestRejectWrongAnswer(t *testing.T) {
	s := New([]byte("test-secret"), time.Minute)
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(ch.Token, "99999"); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestRejectTampered(t *testing.T) {
	s := New([]byte("test-secret"), time.Minute)
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(ch.Token+"x", "1"); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestExpired(t *testing.T) {
	s := New([]byte("test-secret"), 20*time.Millisecond)
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	var a, b int
	_, _ = fmt.Sscanf(ch.Question, "%d + %d", &a, &b)
	if err := s.Verify(ch.Token, fmt.Sprintf("%d", a+b)); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestMissingFields(t *testing.T) {
	s := New([]byte("test-secret"), time.Minute)
	if err := s.Verify("", "1"); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	ch, err := s.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(ch.Token, ""); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
