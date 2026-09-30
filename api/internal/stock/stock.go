package stock

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
)

type Movement struct {
	ID           int64       `json:"id"`
	FoodID       int64       `json:"food_id"`
	FoodName     string      `json:"food_name,omitempty"`
	FoodUnit     string      `json:"food_unit,omitempty"`
	MovementType string      `json:"movement_type"`
	Quantity     quantity.Q  `json:"quantity"`
	Note         *string     `json:"note,omitempty"`
	CreatedBy    *int64      `json:"created_by,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

type Balance struct {
	FoodID       int64      `json:"food_id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Unit         string     `json:"unit"`
	CategoryID   int64      `json:"category_id,omitempty"`
	CategoryCode string     `json:"category_code,omitempty"`
	CategoryName string     `json:"category_name,omitempty"`
	Quantity     quantity.Q `json:"quantity"`
}

type Repository struct {
	DB                 *sql.DB
	AllowNegativeStock bool
}

func (r *Repository) Balances(ctx context.Context) ([]Balance, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit,
			COALESCE(f.category_id, 0), COALESCE(c.code, ''), COALESCE(c.name, ''),
			COALESCE(SUM(CASE WHEN m.movement_type = 'IN' THEN m.quantity
			                  WHEN m.movement_type = 'OUT' THEN -m.quantity
			                  ELSE 0 END), 0) AS qty
		FROM foods f
		LEFT JOIN item_categories c ON c.id = f.category_id
		LEFT JOIN stock_movements m ON m.food_id = f.id
		WHERE f.active = TRUE
		GROUP BY f.id, f.code, f.name, f.unit, f.category_id, c.code, c.name, c.sort_order, f.sort_order
		ORDER BY COALESCE(c.sort_order, 999), f.sort_order ASC, f.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.FoodID, &b.Code, &b.Name, &b.Unit, &b.CategoryID, &b.CategoryCode, &b.CategoryName, &b.Quantity); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) BalanceForFood(ctx context.Context, tx *sql.Tx, foodID int64) (quantity.Q, error) {
	var q quantity.Q
	row := queryRow(ctx, r.DB, tx, `
		SELECT COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN quantity
		                         WHEN movement_type = 'OUT' THEN -quantity
		                         ELSE 0 END), 0)
		FROM stock_movements WHERE food_id = $1`, foodID)
	if err := row.Scan(&q); err != nil {
		return quantity.Zero(), err
	}
	return q, nil
}

func (r *Repository) MoveIn(ctx context.Context, foodID int64, qty quantity.Q, note *string, userID *int64) (Movement, error) {
	if qty.Sign() <= 0 {
		return Movement{}, ErrInvalidQuantity
	}
	return r.insert(ctx, foodID, "IN", qty, note, userID)
}

func (r *Repository) MoveOut(ctx context.Context, foodID int64, qty quantity.Q, note *string, userID *int64) (Movement, error) {
	if qty.Sign() <= 0 {
		return Movement{}, ErrInvalidQuantity
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return Movement{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Lock food row to serialize stock checks.
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM foods WHERE id = $1 FOR UPDATE`, foodID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movement{}, errors.New("food not found")
		}
		return Movement{}, err
	}

	balance, err := r.BalanceForFood(ctx, tx, foodID)
	if err != nil {
		return Movement{}, err
	}
	if !r.AllowNegativeStock && balance.Less(qty) {
		return Movement{}, ErrInsufficientStock
	}

	var m Movement
	err = tx.QueryRowContext(ctx, `
		INSERT INTO stock_movements(food_id, movement_type, quantity, note, created_by)
		VALUES ($1, 'OUT', $2, $3, $4)
		RETURNING id, food_id, movement_type, quantity, note, created_by, created_at`,
		foodID, qty, note, userID,
	).Scan(&m.ID, &m.FoodID, &m.MovementType, &m.Quantity, &m.Note, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		return Movement{}, err
	}
	if err := tx.Commit(); err != nil {
		return Movement{}, err
	}
	return m, nil
}

func (r *Repository) insert(ctx context.Context, foodID int64, typ string, qty quantity.Q, note *string, userID *int64) (Movement, error) {
	var m Movement
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO stock_movements(food_id, movement_type, quantity, note, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, food_id, movement_type, quantity, note, created_by, created_at`,
		foodID, typ, qty, note, userID,
	).Scan(&m.ID, &m.FoodID, &m.MovementType, &m.Quantity, &m.Note, &m.CreatedBy, &m.CreatedAt)
	return m, err
}

func (r *Repository) ListMovements(ctx context.Context, limit int) ([]Movement, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT m.id, m.food_id, f.name, f.unit, m.movement_type, m.quantity, m.note, m.created_by, m.created_at
		FROM stock_movements m
		JOIN foods f ON f.id = m.food_id
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Movement
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.FoodID, &m.FoodName, &m.FoodUnit, &m.MovementType, &m.Quantity, &m.Note, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type rowScanner interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func queryRow(ctx context.Context, db *sql.DB, tx *sql.Tx, query string, args ...any) *sql.Row {
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return db.QueryRowContext(ctx, query, args...)
}
