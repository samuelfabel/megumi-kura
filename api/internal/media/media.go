package media

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

const MaxBytes = 1_048_576 // 1 MiB

var (
	ErrTooLarge    = errors.New("file exceeds 1MB limit")
	ErrInvalidType = errors.New("unsupported content type")
	ErrNotFound    = errors.New("media not found")
	ErrEmpty       = errors.New("empty file")
)

var allowedTypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/webp":    true,
	"image/gif":     true,
	"image/svg+xml": true,
}

type Asset struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int    `json:"size_bytes"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) Create(ctx context.Context, kind, filename, contentType string, data []byte) (Asset, error) {
	if len(data) == 0 {
		return Asset{}, ErrEmpty
	}
	if len(data) > MaxBytes {
		return Asset{}, ErrTooLarge
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "image/jpg" {
		ct = "image/jpeg"
	}
	if !allowedTypes[ct] {
		return Asset{}, ErrInvalidType
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "other"
	}
	var a Asset
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO media_assets(kind, filename, content_type, size_bytes, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, kind, filename, content_type, size_bytes`,
		kind, filename, ct, len(data), data,
	).Scan(&a.ID, &a.Kind, &a.Filename, &a.ContentType, &a.SizeBytes)
	return a, err
}

func (r *Repository) GetData(ctx context.Context, id int64) (contentType string, data []byte, err error) {
	err = r.DB.QueryRowContext(ctx, `
		SELECT content_type, data FROM media_assets WHERE id = $1`, id,
	).Scan(&contentType, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	return contentType, data, err
}
