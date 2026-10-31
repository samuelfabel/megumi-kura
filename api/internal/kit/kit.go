package kit

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
)

var ErrNotFound = errors.New("kit not found")

type Item struct {
	FoodID   int64      `json:"food_id"`
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Unit     string     `json:"unit"`
	Quantity quantity.Q `json:"quantity"`
	IconID   *int64     `json:"icon_media_id,omitempty"`
}

type Kit struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	IconMediaID  *int64    `json:"icon_media_id,omitempty"`
	SortOrder    int       `json:"sort_order"`
	Active       bool      `json:"active"`
	Items        []Item    `json:"items,omitempty"`
	Available    int64     `json:"available,omitempty"`
	LimitingCode string    `json:"limiting_food_code,omitempty"`
	LimitingName string    `json:"limiting_food_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IconMediaID *int64 `json:"icon_media_id"`
}

type PutItem struct {
	FoodID   int64      `json:"food_id"`
	Quantity quantity.Q `json:"quantity"`
}

type Repository struct {
	DB    *sql.DB
	Stock *stock.Repository
}

func (r *Repository) List(ctx context.Context, activeOnly bool) ([]Kit, error) {
	q := `
		SELECT id, code, name, description, icon_media_id, sort_order, active, created_at, updated_at
		FROM kits`
	if activeOnly {
		q += ` WHERE active = TRUE`
	}
	q += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Kit
	for rows.Next() {
		var k Kit
		var icon sql.NullInt64
		if err := rows.Scan(&k.ID, &k.Code, &k.Name, &k.Description, &icon, &k.SortOrder, &k.Active, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		if icon.Valid {
			k.IconMediaID = &icon.Int64
		}
		items, err := r.listItems(ctx, k.ID)
		if err != nil {
			return nil, err
		}
		k.Items = items
		if err := r.fillAvailability(ctx, &k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Kit, error) {
	var k Kit
	var icon sql.NullInt64
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, code, name, description, icon_media_id, sort_order, active, created_at, updated_at
		FROM kits WHERE id = $1`, id,
	).Scan(&k.ID, &k.Code, &k.Name, &k.Description, &icon, &k.SortOrder, &k.Active, &k.CreatedAt, &k.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Kit{}, ErrNotFound
	}
	if err != nil {
		return Kit{}, err
	}
	if icon.Valid {
		k.IconMediaID = &icon.Int64
	}
	k.Items, err = r.listItems(ctx, k.ID)
	if err != nil {
		return Kit{}, err
	}
	if err := r.fillAvailability(ctx, &k); err != nil {
		return Kit{}, err
	}
	return k, nil
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Kit, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if in.Code == "" || in.Name == "" {
		return Kit{}, errors.New("code and name are required")
	}
	var id int64
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO kits(code, name, description, sort_order, icon_media_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		in.Code, in.Name, strings.TrimSpace(in.Description), in.SortOrder, in.IconMediaID,
	).Scan(&id)
	if err != nil {
		return Kit{}, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, in CreateInput, active *bool) (Kit, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if in.Code == "" || in.Name == "" {
		return Kit{}, errors.New("code and name are required")
	}
	act := true
	if active != nil {
		act = *active
	} else {
		existing, err := r.Get(ctx, id)
		if err != nil {
			return Kit{}, err
		}
		act = existing.Active
	}
	res, err := r.DB.ExecContext(ctx, `
		UPDATE kits SET code=$2, name=$3, description=$4, sort_order=$5, icon_media_id=$6,
		active=$7, updated_at=NOW() WHERE id=$1`,
		id, in.Code, in.Name, strings.TrimSpace(in.Description), in.SortOrder, in.IconMediaID, act)
	if err != nil {
		return Kit{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Kit{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *Repository) ReplaceItems(ctx context.Context, kitID int64, items []PutItem) (Kit, error) {
	if _, err := r.Get(ctx, kitID); err != nil {
		return Kit{}, err
	}
	if len(items) == 0 {
		return Kit{}, errors.New("kit must contain at least one item")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return Kit{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM kit_items WHERE kit_id = $1`, kitID); err != nil {
		return Kit{}, err
	}
	for _, it := range items {
		if it.FoodID == 0 || it.Quantity.Sign() <= 0 {
			return Kit{}, errors.New("each kit item needs food_id and quantity > 0")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO kit_items(kit_id, food_id, quantity) VALUES ($1, $2, $3)`,
			kitID, it.FoodID, it.Quantity); err != nil {
			return Kit{}, err
		}
	}
	// Keep legacy basket_items in sync for default kit
	var code string
	if err := tx.QueryRowContext(ctx, `SELECT code FROM kits WHERE id = $1`, kitID).Scan(&code); err != nil {
		return Kit{}, err
	}
	if code == "cesta_basica" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM basket_items`); err != nil {
			return Kit{}, err
		}
		for _, it := range items {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO basket_items(food_id, quantity) VALUES ($1, $2)`, it.FoodID, it.Quantity); err != nil {
				return Kit{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Kit{}, err
	}
	return r.Get(ctx, kitID)
}

func (r *Repository) listItems(ctx context.Context, kitID int64) ([]Item, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, ki.quantity, f.icon_media_id
		FROM kit_items ki
		JOIN foods f ON f.id = ki.food_id
		WHERE ki.kit_id = $1
		ORDER BY f.sort_order ASC, f.name ASC`, kitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		var icon sql.NullInt64
		if err := rows.Scan(&it.FoodID, &it.Code, &it.Name, &it.Unit, &it.Quantity, &icon); err != nil {
			return nil, err
		}
		if icon.Valid {
			it.IconID = &icon.Int64
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) fillAvailability(ctx context.Context, k *Kit) error {
	if len(k.Items) == 0 || r.Stock == nil {
		return nil
	}
	balances, err := r.Stock.Balances(ctx)
	if err != nil {
		return err
	}
	balByFood := map[int64]quantity.Q{}
	for _, b := range balances {
		balByFood[b.FoodID] = b.Quantity
	}
	minAvail := int64(math.MaxInt64)
	for _, item := range k.Items {
		n := balByFood[item.FoodID].DivFloorInt(item.Quantity)
		if n < minAvail {
			minAvail = n
			k.LimitingCode = item.Code
			k.LimitingName = item.Name
		}
	}
	if minAvail == int64(math.MaxInt64) || minAvail < 0 {
		minAvail = 0
	}
	k.Available = minAvail
	return nil
}

// ExpandItems returns kit composition quantities (for delivery / promise).
func (r *Repository) ExpandItems(ctx context.Context, kitID int64) ([]PutItem, error) {
	items, err := r.listItems(ctx, kitID)
	if err != nil {
		return nil, err
	}
	out := make([]PutItem, 0, len(items))
	for _, it := range items {
		out = append(out, PutItem{FoodID: it.FoodID, Quantity: it.Quantity})
	}
	return out, nil
}
