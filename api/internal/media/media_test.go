package media

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func TestCreateValidation(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	cases := []struct {
		name string
		ct   string
		data []byte
		want error
	}{
		{"empty", "image/png", nil, ErrEmpty},
		{"too large", "image/png", make([]byte, MaxBytes+1), ErrTooLarge},
		{"unsupported type", "application/pdf", []byte("x"), ErrInvalidType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := r.Create(ctx, "food_icon", "f", tc.ct, tc.data); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreateAndGetData(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	payload := []byte{0x89, 'P', 'N', 'G'}

	a, err := r.Create(ctx, "  ", "icone.jpg", " IMAGE/JPG ", payload)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Kind != "other" || a.ContentType != "image/jpeg" || a.SizeBytes != len(payload) {
		t.Fatalf("unexpected asset: %+v", a)
	}

	ct, data, err := r.GetData(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ct != "image/jpeg" || !bytes.Equal(data, payload) {
		t.Fatalf("got %q %v", ct, data)
	}
	if _, _, err := r.GetData(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing asset err = %v, want ErrNotFound", err)
	}
}
