package basket

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
)

type Item struct {
	FoodID   int64      `json:"food_id"`
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Unit     string     `json:"unit"`
	Quantity quantity.Q `json:"quantity"`
}

type Composition struct {
	Items             []Item `json:"items"`
	AvailableBaskets  int64  `json:"available_baskets"`
	LimitingFoodCode  string `json:"limiting_food_code,omitempty"`
	LimitingFoodName  string `json:"limiting_food_name,omitempty"`
}

type PutItem struct {
	FoodID   int64      `json:"food_id"`
	Quantity quantity.Q `json:"quantity"`
}

type Repository struct {
	DB    *sql.DB
	Stock *stock.Repository
}

func (r *Repository) Get(ctx context.Context) (Composition, error) {
	items, err := r.listItems(ctx)
	if err != nil {
		return Composition{}, err
	}
	balances, err := r.Stock.Balances(ctx)
	if err != nil {
		return Composition{}, err
	}
	balByFood := map[int64]quantity.Q{}
	for _, b := range balances {
		balByFood[b.FoodID] = b.Quantity
	}

	comp := Composition{Items: items}
	if len(items) == 0 {
		return comp, nil
	}

	minBaskets := int64(math.MaxInt64)
	for _, item := range items {
		bal := balByFood[item.FoodID]
		n := bal.DivFloorInt(item.Quantity)
		if n < minBaskets {
			minBaskets = n
			comp.LimitingFoodCode = item.Code
			comp.LimitingFoodName = item.Name
		}
	}
	if minBaskets == int64(math.MaxInt64) || minBaskets < 0 {
		minBaskets = 0
	}
	comp.AvailableBaskets = minBaskets
	return comp, nil
}

func (r *Repository) listItems(ctx context.Context) ([]Item, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.code, f.name, f.unit, bi.quantity
		FROM basket_items bi
		JOIN foods f ON f.id = bi.food_id
		ORDER BY f.sort_order ASC, f.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.FoodID, &it.Code, &it.Name, &it.Unit, &it.Quantity); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) Replace(ctx context.Context, items []PutItem) (Composition, error) {
	if len(items) == 0 {
		return Composition{}, errors.New("basket must contain at least one item")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return Composition{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM basket_items`); err != nil {
		return Composition{}, err
	}
	for _, it := range items {
		if it.FoodID == 0 || it.Quantity.Sign() <= 0 {
			return Composition{}, errors.New("each basket item needs food_id and quantity > 0")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO basket_items(food_id, quantity) VALUES ($1, $2)`, it.FoodID, it.Quantity); err != nil {
			return Composition{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Composition{}, err
	}
	return r.Get(ctx)
}
