package person

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

var ErrNotFound = errors.New("person not found")

type Person struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"full_name"`
	Notes     string    `json:"notes"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Need struct {
	ID        int64      `json:"id"`
	PersonID  int64      `json:"person_id"`
	FoodID    int64      `json:"food_id"`
	FoodName  string     `json:"food_name,omitempty"`
	FoodUnit  string     `json:"food_unit,omitempty"`
	Category  string     `json:"category_name,omitempty"`
	Quantity  quantity.Q `json:"quantity"`
	NeedKind  string     `json:"need_kind"` // sporadic | recurring
	Frequency *string    `json:"frequency,omitempty"`
	Notes     string     `json:"notes"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreatePersonInput struct {
	FullName string `json:"full_name"`
	Notes    string `json:"notes"`
}

type CreateNeedInput struct {
	PersonID  int64      `json:"person_id"`
	FoodID    int64      `json:"food_id"`
	Quantity  quantity.Q `json:"quantity"`
	NeedKind  string     `json:"need_kind"`
	Frequency *string    `json:"frequency"`
	Notes     string     `json:"notes"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) List(ctx context.Context) ([]Person, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, full_name, notes, active, created_at, updated_at
		FROM people
		ORDER BY full_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.FullName, &p.Notes, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreatePersonInput) (Person, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" {
		return Person{}, errors.New("full_name is required")
	}
	var p Person
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO people(full_name, notes)
		VALUES ($1, $2)
		RETURNING id, full_name, notes, active, created_at, updated_at`,
		in.FullName, strings.TrimSpace(in.Notes),
	).Scan(&p.ID, &p.FullName, &p.Notes, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) Update(ctx context.Context, id int64, in CreatePersonInput, active bool) (Person, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" {
		return Person{}, errors.New("full_name is required")
	}
	var p Person
	err := r.DB.QueryRowContext(ctx, `
		UPDATE people
		SET full_name = $2, notes = $3, active = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, full_name, notes, active, created_at, updated_at`,
		id, in.FullName, strings.TrimSpace(in.Notes), active,
	).Scan(&p.ID, &p.FullName, &p.Notes, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Person{}, ErrNotFound
	}
	return p, err
}

func (r *Repository) ListNeeds(ctx context.Context, personID *int64) ([]Need, error) {
	query := `
		SELECT pn.id, pn.person_id, pn.food_id, f.name, f.unit, COALESCE(c.name, ''),
		       pn.quantity, pn.need_kind, pn.frequency, pn.notes, pn.active,
		       pn.created_at, pn.updated_at
		FROM person_needs pn
		JOIN foods f ON f.id = pn.food_id
		LEFT JOIN item_categories c ON c.id = f.category_id`
	var args []any
	if personID != nil {
		query += ` WHERE pn.person_id = $1`
		args = append(args, *personID)
	}
	query += ` ORDER BY pn.person_id, f.sort_order, pn.id`

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Need
	for rows.Next() {
		var n Need
		if err := rows.Scan(&n.ID, &n.PersonID, &n.FoodID, &n.FoodName, &n.FoodUnit, &n.Category,
			&n.Quantity, &n.NeedKind, &n.Frequency, &n.Notes, &n.Active, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) CreateNeed(ctx context.Context, in CreateNeedInput) (Need, error) {
	if in.Frequency != nil && strings.TrimSpace(*in.Frequency) == "" {
		in.Frequency = nil
	}
	if err := validateNeed(in.NeedKind, in.Frequency); err != nil {
		return Need{}, err
	}
	if in.PersonID == 0 || in.FoodID == 0 || in.Quantity.Sign() <= 0 {
		return Need{}, errors.New("person_id, food_id and quantity are required")
	}
	var n Need
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO person_needs(person_id, food_id, quantity, need_kind, frequency, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, person_id, food_id, quantity, need_kind, frequency, notes, active, created_at, updated_at`,
		in.PersonID, in.FoodID, in.Quantity, in.NeedKind, in.Frequency, strings.TrimSpace(in.Notes),
	).Scan(&n.ID, &n.PersonID, &n.FoodID, &n.Quantity, &n.NeedKind, &n.Frequency, &n.Notes, &n.Active, &n.CreatedAt, &n.UpdatedAt)
	return n, err
}

func (r *Repository) DeleteNeed(ctx context.Context, id int64) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM person_needs WHERE id = $1`, id)
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

func validateNeed(kind string, freq *string) error {
	switch kind {
	case "sporadic":
		if freq != nil && *freq != "" {
			return errors.New("sporadic needs must not have frequency")
		}
		return nil
	case "recurring":
		if freq == nil || *freq == "" {
			return errors.New("recurring needs require frequency")
		}
		switch *freq {
		case "weekly", "biweekly", "monthly":
			return nil
		default:
			return errors.New("frequency must be weekly, biweekly or monthly")
		}
	default:
		return errors.New("need_kind must be sporadic or recurring")
	}
}
