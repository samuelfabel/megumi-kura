package delivery

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
)

var ErrNotFound = errors.New("delivery not found")

type Item struct {
	FoodID   int64      `json:"food_id"`
	FoodName string     `json:"food_name,omitempty"`
	FoodUnit string     `json:"food_unit,omitempty"`
	Quantity quantity.Q `json:"quantity"`
}

type Delivery struct {
	ID          int64     `json:"id"`
	PersonID    int64     `json:"person_id"`
	PersonName  string    `json:"person_name,omitempty"`
	DeliveredOn string    `json:"delivered_on"`
	Note        string    `json:"note"`
	DeductStock bool      `json:"deduct_stock"`
	CreatedBy   *int64    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Items       []Item    `json:"items,omitempty"`
}

type CreateInput struct {
	PersonID    int64  `json:"person_id"`
	DeliveredOn string `json:"delivered_on"`
	Note        string `json:"note"`
	DeductStock *bool  `json:"deduct_stock"`
	Items       []Item `json:"items"`
}

type Repository struct {
	DB                 *sql.DB
	AllowNegativeStock bool
}

func (r *Repository) List(ctx context.Context, limit int) ([]Delivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT d.id, d.person_id, p.full_name, to_char(d.delivered_on, 'YYYY-MM-DD'),
		       d.note, d.deduct_stock, d.created_by, d.created_at
		FROM deliveries d
		JOIN people p ON p.id = d.person_id
		ORDER BY d.delivered_on DESC, d.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.PersonID, &d.PersonName, &d.DeliveredOn, &d.Note, &d.DeductStock, &d.CreatedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		items, err := r.itemsFor(ctx, nil, d.ID)
		if err != nil {
			return nil, err
		}
		d.Items = items
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) itemsFor(ctx context.Context, tx *sql.Tx, deliveryID int64) ([]Item, error) {
	q := `
		SELECT di.food_id, f.name, f.unit, di.quantity
		FROM delivery_items di
		JOIN foods f ON f.id = di.food_id
		WHERE di.delivery_id = $1
		ORDER BY f.sort_order, di.id`
	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(ctx, q, deliveryID)
	} else {
		rows, err = r.DB.QueryContext(ctx, q, deliveryID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.FoodID, &it.FoodName, &it.FoodUnit, &it.Quantity); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput, userID *int64) (Delivery, error) {
	if in.PersonID == 0 || len(in.Items) == 0 {
		return Delivery{}, errors.New("person_id and at least one item are required")
	}
	if strings.TrimSpace(in.DeliveredOn) == "" {
		in.DeliveredOn = time.Now().Format("2006-01-02")
	}
	deduct := true
	if in.DeductStock != nil {
		deduct = *in.DeductStock
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return Delivery{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var personName string
	if err := tx.QueryRowContext(ctx, `SELECT full_name FROM people WHERE id = $1`, in.PersonID).Scan(&personName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Delivery{}, errors.New("person not found")
		}
		return Delivery{}, err
	}

	if deduct && !r.AllowNegativeStock {
		for _, it := range in.Items {
			if it.FoodID == 0 || it.Quantity.Sign() <= 0 {
				return Delivery{}, errors.New("each item needs food_id and quantity > 0")
			}
			var bal quantity.Q
			if err := tx.QueryRowContext(ctx, `
				SELECT COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN quantity
				                         WHEN movement_type = 'OUT' THEN -quantity ELSE 0 END), 0)
				FROM stock_movements WHERE food_id = $1`, it.FoodID).Scan(&bal); err != nil {
				return Delivery{}, err
			}
			if bal.Less(it.Quantity) {
				return Delivery{}, stock.ErrInsufficientStock
			}
		}
	}

	var d Delivery
	err = tx.QueryRowContext(ctx, `
		INSERT INTO deliveries(person_id, delivered_on, note, deduct_stock, created_by)
		VALUES ($1, $2::date, $3, $4, $5)
		RETURNING id, person_id, to_char(delivered_on, 'YYYY-MM-DD'), note, deduct_stock, created_by, created_at`,
		in.PersonID, in.DeliveredOn, strings.TrimSpace(in.Note), deduct, userID,
	).Scan(&d.ID, &d.PersonID, &d.DeliveredOn, &d.Note, &d.DeductStock, &d.CreatedBy, &d.CreatedAt)
	if err != nil {
		return Delivery{}, err
	}
	d.PersonName = personName

	note := "delivery #" + strconv.FormatInt(d.ID, 10)
	for _, it := range in.Items {
		if it.FoodID == 0 || it.Quantity.Sign() <= 0 {
			return Delivery{}, errors.New("each item needs food_id and quantity > 0")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO delivery_items(delivery_id, food_id, quantity)
			VALUES ($1, $2, $3)`, d.ID, it.FoodID, it.Quantity); err != nil {
			return Delivery{}, err
		}
		if deduct {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO stock_movements(food_id, movement_type, quantity, note, created_by)
				VALUES ($1, 'OUT', $2, $3, $4)`, it.FoodID, it.Quantity, note, userID); err != nil {
				return Delivery{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return Delivery{}, err
	}

	items, err := r.itemsFor(ctx, nil, d.ID)
	if err != nil {
		return d, err
	}
	d.Items = items
	return d, nil
}
