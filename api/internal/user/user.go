package user

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("user not found")

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Active       bool      `json:"active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, username, password_hash, display_name, active, created_at, updated_at
		FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Active, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) FindByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, username, password_hash, display_name, active, created_at, updated_at
		FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Active, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
