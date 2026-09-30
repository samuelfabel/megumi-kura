package category

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("category not found")

type Category struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, code, name, sort_order, active, created_at, updated_at
		FROM item_categories
		ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Category, error) {
	in.Code = strings.TrimSpace(strings.ToLower(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Code == "" || in.Name == "" {
		return Category{}, errors.New("code and name are required")
	}
	var c Category
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO item_categories(code, name, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, code, name, sort_order, active, created_at, updated_at`,
		in.Code, in.Name, in.SortOrder,
	).Scan(&c.ID, &c.Code, &c.Name, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) Update(ctx context.Context, id int64, in CreateInput) (Category, error) {
	in.Code = strings.TrimSpace(strings.ToLower(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Code == "" || in.Name == "" {
		return Category{}, errors.New("code and name are required")
	}
	var c Category
	err := r.DB.QueryRowContext(ctx, `
		UPDATE item_categories
		SET code = $2, name = $3, sort_order = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, code, name, sort_order, active, created_at, updated_at`,
		id, in.Code, in.Name, in.SortOrder,
	).Scan(&c.ID, &c.Code, &c.Name, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return c, err
}
