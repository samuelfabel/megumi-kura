package food

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("food not found")

type Food struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	CategoryID   *int64    `json:"category_id,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
	CategoryCode string    `json:"category_code,omitempty"`
	SortOrder    int       `json:"sort_order"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateInput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Unit       string `json:"unit"`
	CategoryID *int64 `json:"category_id"`
	SortOrder  int    `json:"sort_order"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) List(ctx context.Context) ([]Food, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, f.category_id, COALESCE(c.name, ''), COALESCE(c.code, ''),
		       f.sort_order, f.active, f.created_at, f.updated_at
		FROM foods f
		LEFT JOIN item_categories c ON c.id = f.category_id
		ORDER BY COALESCE(c.sort_order, 999), f.sort_order ASC, f.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Food
	for rows.Next() {
		var f Food
		if err := rows.Scan(&f.ID, &f.Code, &f.Name, &f.Unit, &f.CategoryID, &f.CategoryName, &f.CategoryCode,
			&f.SortOrder, &f.Active, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Food, error) {
	in.Code = strings.TrimSpace(strings.ToLower(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Unit = strings.TrimSpace(in.Unit)
	if in.Code == "" || in.Name == "" || in.Unit == "" {
		return Food{}, errors.New("code, name and unit are required")
	}
	var f Food
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO foods(code, name, unit, sort_order, category_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, code, name, unit, category_id, sort_order, active, created_at, updated_at`,
		in.Code, in.Name, in.Unit, in.SortOrder, in.CategoryID,
	).Scan(&f.ID, &f.Code, &f.Name, &f.Unit, &f.CategoryID, &f.SortOrder, &f.Active, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Food, error) {
	var f Food
	err := r.DB.QueryRowContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, f.category_id, COALESCE(c.name, ''), COALESCE(c.code, ''),
		       f.sort_order, f.active, f.created_at, f.updated_at
		FROM foods f
		LEFT JOIN item_categories c ON c.id = f.category_id
		WHERE f.id = $1`, id).Scan(
		&f.ID, &f.Code, &f.Name, &f.Unit, &f.CategoryID, &f.CategoryName, &f.CategoryCode,
		&f.SortOrder, &f.Active, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Food{}, ErrNotFound
	}
	return f, err
}
