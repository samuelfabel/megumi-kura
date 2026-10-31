package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func TestHashPassword(t *testing.T) {
	h, err := HashPassword("s3nha-forte")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if h == "s3nha-forte" {
		t.Fatal("hash equals plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h), []byte("s3nha-forte")); err != nil {
		t.Fatalf("hash does not verify: %v", err)
	}
}

func TestCreateValidatesAndFiltersPermissions(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()

	if _, err := r.Create(ctx, CreateInput{Username: "ana", Password: "x"}); err == nil {
		t.Fatal("expected error without email")
	}

	u, err := r.Create(ctx, CreateInput{
		Username:    " ana ",
		Email:       " ana@example.org ",
		DisplayName: "Ana",
		Password:    "s3nha",
		Permissions: []string{"stock", "kits", "root", "stock"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.Username != "ana" || u.Email == nil || *u.Email != "ana@example.org" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.IsBootstrapAdmin {
		t.Fatal("created user must not be bootstrap admin")
	}
	if len(u.Permissions) != 2 || u.Permissions[0] != "kits" || u.Permissions[1] != "stock" {
		t.Fatalf("permissions = %v, want [kits stock]", u.Permissions)
	}

	ok, err := r.HasPermission(ctx, u, "stock")
	if err != nil || !ok {
		t.Fatalf("HasPermission(stock) = %v, %v; want true", ok, err)
	}
	ok, err = r.HasPermission(ctx, u, "users")
	if err != nil || ok {
		t.Fatalf("HasPermission(users) = %v, %v; want false", ok, err)
	}

	byEmail, err := r.FindByEmail(ctx, "ANA@example.org")
	if err != nil || byEmail.ID != u.ID {
		t.Fatalf("FindByEmail case-insensitive = %+v, %v", byEmail, err)
	}
}

func TestBootstrapAdminHasAllPermissionsAndCannotBeDeactivated(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	var id int64
	if err := r.DB.QueryRow(`
		INSERT INTO users(username, password_hash, is_bootstrap_admin) VALUES ('admin', $1, TRUE) RETURNING id`,
		hash).Scan(&id); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	admin, err := r.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("find admin: %v", err)
	}
	for _, p := range AllPermissions {
		ok, err := r.HasPermission(ctx, admin, p)
		if err != nil || !ok {
			t.Fatalf("bootstrap admin HasPermission(%s) = %v, %v", p, ok, err)
		}
	}
	if err := r.SetActive(ctx, id, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deactivate bootstrap admin err = %v, want ErrNotFound", err)
	}
}

func TestPasswordResetToken(t *testing.T) {
	r := &Repository{DB: testdb.Open(t)}
	ctx := context.Background()
	u, err := r.Create(ctx, CreateInput{Username: "bia", Email: "bia@example.org", Password: "antiga"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	raw, err := r.CreateResetToken(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := r.ConsumeResetToken(ctx, raw, "nova-senha"); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if err := r.ConsumeResetToken(ctx, raw, "outra"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reuse token err = %v, want ErrNotFound", err)
	}

	got, err := r.FindByUsername(ctx, "bia")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("nova-senha")); err != nil {
		t.Fatal("password was not updated by reset")
	}

	expired, err := r.CreateResetToken(ctx, u.ID, -time.Minute)
	if err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	if err := r.ConsumeResetToken(ctx, expired, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired token err = %v, want ErrNotFound", err)
	}
}
