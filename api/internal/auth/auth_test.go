package auth

import "testing"

func TestHashAndCompare(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "admin123" || hash == "" {
		t.Fatal("hash must not be plaintext")
	}
	s := &Service{Secret: []byte("test-secret"), SessionHours: 1}
	token, err := s.issueToken(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.parseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("got %d want 42", id)
	}
}

func TestParseRejectsTampered(t *testing.T) {
	s := &Service{Secret: []byte("test-secret"), SessionHours: 1}
	token, err := s.issueToken(7)
	if err != nil {
		t.Fatal(err)
	}
	bad := token + "x"
	if _, err := s.parseToken(bad); err == nil {
		t.Fatal("expected error for tampered token")
	}
}
