package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("user not found")

var AllPermissions = []string{
	"stock", "kits", "people", "deliveries", "promises",
	"social", "email", "users", "settings", "categories", "needs",
}

type User struct {
	ID               int64     `json:"id"`
	Username         string    `json:"username"`
	Email            *string   `json:"email,omitempty"`
	DisplayName      string    `json:"display_name"`
	Active           bool      `json:"active"`
	IsBootstrapAdmin bool      `json:"is_bootstrap_admin"`
	Permissions      []string  `json:"permissions,omitempty"`
	PasswordHash     string    `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateInput struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Password    string   `json:"password"`
	Permissions []string `json:"permissions"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (User, error) {
	return r.scanOne(ctx, `
		SELECT id, username, email, password_hash, display_name, active, is_bootstrap_admin, created_at, updated_at
		FROM users WHERE username = $1`, username)
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return r.scanOne(ctx, `
		SELECT id, username, email, password_hash, display_name, active, is_bootstrap_admin, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)`, strings.TrimSpace(email))
}

func (r *Repository) FindByID(ctx context.Context, id int64) (User, error) {
	u, err := r.scanOne(ctx, `
		SELECT id, username, email, password_hash, display_name, active, is_bootstrap_admin, created_at, updated_at
		FROM users WHERE id = $1`, id)
	if err != nil {
		return User{}, err
	}
	perms, err := r.ListPermissions(ctx, id)
	if err != nil {
		return User{}, err
	}
	u.Permissions = perms
	return u, nil
}

func (r *Repository) scanOne(ctx context.Context, q string, arg any) (User, error) {
	var u User
	var email sql.NullString
	err := r.DB.QueryRowContext(ctx, q, arg).Scan(
		&u.ID, &u.Username, &email, &u.PasswordHash, &u.DisplayName, &u.Active, &u.IsBootstrapAdmin, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if email.Valid && email.String != "" {
		u.Email = &email.String
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, username, email, display_name, active, is_bootstrap_admin, created_at, updated_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var email sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &email, &u.DisplayName, &u.Active, &u.IsBootstrapAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		if email.Valid && email.String != "" {
			u.Email = &email.String
		}
		perms, err := r.ListPermissions(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		u.Permissions = perms
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) ListPermissions(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT permission FROM user_permissions WHERE user_id = $1 ORDER BY permission`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) HasPermission(ctx context.Context, u User, perm string) (bool, error) {
	if u.IsBootstrapAdmin {
		return true, nil
	}
	var n int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM user_permissions WHERE user_id = $1 AND permission = $2`, u.ID, perm).Scan(&n)
	return n > 0, err
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (User, error) {
	in.Username = strings.TrimSpace(in.Username)
	in.Email = strings.TrimSpace(in.Email)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.Username == "" || in.Email == "" || in.Password == "" {
		return User{}, errors.New("username, email and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	var id int64
	err = r.DB.QueryRowContext(ctx, `
		INSERT INTO users(username, email, password_hash, display_name, is_bootstrap_admin)
		VALUES ($1, $2, $3, $4, FALSE) RETURNING id`,
		in.Username, in.Email, string(hash), in.DisplayName,
	).Scan(&id)
	if err != nil {
		return User{}, err
	}
	if err := r.ReplacePermissions(ctx, id, in.Permissions); err != nil {
		return User{}, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) ReplacePermissions(ctx context.Context, userID int64, perms []string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_permissions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, p := range AllPermissions {
		allowed[p] = true
	}
	for _, p := range perms {
		if !allowed[p] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_permissions(user_id, permission) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, userID, p); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) SetActive(ctx context.Context, id int64, active bool) error {
	res, err := r.DB.ExecContext(ctx, `UPDATE users SET active=$2, updated_at=NOW() WHERE id=$1 AND is_bootstrap_admin = FALSE`, id, active)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) SetPassword(ctx context.Context, id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`, id, string(hash))
	return err
}

func (r *Repository) CreateResetToken(ctx context.Context, userID int64, ttl time.Duration) (raw string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	exp := time.Now().Add(ttl)
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO password_reset_tokens(user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, exp)
	return raw, err
}

func (r *Repository) ConsumeResetToken(ctx context.Context, raw, newPassword string) error {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	hash := hex.EncodeToString(sum[:])
	var userID int64
	var id int64
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, user_id FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()`, hash,
	).Scan(&id, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := r.SetPassword(ctx, userID, newPassword); err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
