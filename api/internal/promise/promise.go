package promise

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

var ErrNotFound = errors.New("promise not found")

type Promise struct {
	ID          int64      `json:"id"`
	FoodID      int64      `json:"food_id"`
	FoodName    string     `json:"food_name,omitempty"`
	FoodUnit    string     `json:"food_unit,omitempty"`
	PersonName  string     `json:"person_name"`
	Quantity    quantity.Q `json:"quantity"`
	PromiseDate string     `json:"promise_date"` // YYYY-MM-DD
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Aggregate struct {
	FoodID   int64      `json:"food_id"`
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Unit     string     `json:"unit"`
	Quantity quantity.Q `json:"quantity"`
}

type CreateInput struct {
	FoodID      int64      `json:"food_id"`
	PersonName  string     `json:"person_name"`
	Quantity    quantity.Q `json:"quantity"`
	PromiseDate string     `json:"promise_date"`
}

type Repository struct {
	DB *sql.DB
}

// PublicAggregates returns totals for promises on the given date (inclusive day).
// Expired promises (promise_date < today) are excluded by the caller passing today.
func (r *Repository) PublicAggregates(ctx context.Context, onDate time.Time) ([]Aggregate, error) {
	day := onDate.Format("2006-01-02")
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, COALESCE(SUM(p.quantity), 0) AS qty
		FROM foods f
		JOIN promises p ON p.food_id = f.id AND p.promise_date = $1::date
		WHERE f.active = TRUE
		GROUP BY f.id, f.code, f.name, f.unit, f.sort_order
		HAVING SUM(p.quantity) > 0
		ORDER BY f.sort_order ASC, f.name ASC`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Aggregate
	for rows.Next() {
		var a Aggregate
		if err := rows.Scan(&a.FoodID, &a.Code, &a.Name, &a.Unit, &a.Quantity); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ValidAggregatesFromToday sums promises with promise_date >= today.
func (r *Repository) ValidAggregatesFromToday(ctx context.Context, today time.Time) ([]Aggregate, error) {
	day := today.Format("2006-01-02")
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, COALESCE(SUM(p.quantity), 0) AS qty
		FROM foods f
		JOIN promises p ON p.food_id = f.id AND p.promise_date >= $1::date
		WHERE f.active = TRUE
		GROUP BY f.id, f.code, f.name, f.unit, f.sort_order
		HAVING SUM(p.quantity) > 0
		ORDER BY f.sort_order ASC, f.name ASC`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Aggregate
	for rows.Next() {
		var a Aggregate
		if err := rows.Scan(&a.FoodID, &a.Code, &a.Name, &a.Unit, &a.Quantity); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) ListAdmin(ctx context.Context, fromDate *time.Time) ([]Promise, error) {
	query := `
		SELECT p.id, p.food_id, f.name, f.unit, p.person_name, p.quantity,
		       to_char(p.promise_date, 'YYYY-MM-DD'), p.created_at, p.updated_at
		FROM promises p
		JOIN foods f ON f.id = p.food_id`
	var args []any
	if fromDate != nil {
		query += ` WHERE p.promise_date >= $1::date`
		args = append(args, fromDate.Format("2006-01-02"))
	}
	query += ` ORDER BY p.promise_date ASC, f.sort_order ASC, p.id ASC`

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Promise
	for rows.Next() {
		var p Promise
		if err := rows.Scan(&p.ID, &p.FoodID, &p.FoodName, &p.FoodUnit, &p.PersonName, &p.Quantity, &p.PromiseDate, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Promise, error) {
	in.PersonName = strings.TrimSpace(in.PersonName)
	if in.PersonName == "" || in.FoodID == 0 || in.Quantity.Sign() <= 0 || strings.TrimSpace(in.PromiseDate) == "" {
		return Promise{}, errors.New("food_id, person_name, quantity and promise_date are required")
	}
	var p Promise
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO promises(food_id, person_name, quantity, promise_date)
		VALUES ($1, $2, $3, $4::date)
		RETURNING id, food_id, person_name, quantity, to_char(promise_date, 'YYYY-MM-DD'), created_at, updated_at`,
		in.FoodID, in.PersonName, in.Quantity, in.PromiseDate,
	).Scan(&p.ID, &p.FoodID, &p.PersonName, &p.Quantity, &p.PromiseDate, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM promises WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
