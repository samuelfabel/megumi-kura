package need

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

var ErrNotFound = errors.New("community need not found")

type CommunityNeed struct {
	ID         int64      `json:"id"`
	FoodID     int64      `json:"food_id"`
	FoodName   string     `json:"food_name,omitempty"`
	FoodUnit   string     `json:"food_unit,omitempty"`
	FoodCode   string     `json:"food_code,omitempty"`
	CategoryID int64      `json:"category_id,omitempty"`
	Category   string     `json:"category_name,omitempty"`
	Title      string     `json:"title"`
	Quantity   quantity.Q `json:"quantity"`
	NeedKind   string     `json:"need_kind"`
	Frequency  *string    `json:"frequency,omitempty"`
	Active     bool       `json:"active"`
	StartsOn   string     `json:"starts_on"`
	EndsOn     *string    `json:"ends_on,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateInput struct {
	FoodID    int64      `json:"food_id"`
	Title     string     `json:"title"`
	Quantity  quantity.Q `json:"quantity"`
	NeedKind  string     `json:"need_kind"`
	Frequency *string    `json:"frequency"`
	StartsOn  string     `json:"starts_on"`
	EndsOn    *string    `json:"ends_on"`
	Active    *bool      `json:"active"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) List(ctx context.Context, activeOnly bool) ([]CommunityNeed, error) {
	query := `
		SELECT n.id, n.food_id, f.name, f.unit, f.code, COALESCE(f.category_id, 0), COALESCE(c.name, ''),
		       n.title, n.quantity, n.need_kind, n.frequency, n.active,
		       to_char(n.starts_on, 'YYYY-MM-DD'),
		       CASE WHEN n.ends_on IS NULL THEN NULL ELSE to_char(n.ends_on, 'YYYY-MM-DD') END,
		       n.created_at, n.updated_at
		FROM community_needs n
		JOIN foods f ON f.id = n.food_id
		LEFT JOIN item_categories c ON c.id = f.category_id`
	if activeOnly {
		query += ` WHERE n.active = TRUE
			AND n.starts_on <= CURRENT_DATE
			AND (n.ends_on IS NULL OR n.ends_on >= CURRENT_DATE)`
	}
	query += ` ORDER BY c.sort_order NULLS LAST, f.sort_order, n.id`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommunityNeed
	for rows.Next() {
		var n CommunityNeed
		var ends sql.NullString
		if err := rows.Scan(&n.ID, &n.FoodID, &n.FoodName, &n.FoodUnit, &n.FoodCode, &n.CategoryID, &n.Category,
			&n.Title, &n.Quantity, &n.NeedKind, &n.Frequency, &n.Active, &n.StartsOn, &ends, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if ends.Valid {
			n.EndsOn = &ends.String
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (CommunityNeed, error) {
	if in.Frequency != nil && strings.TrimSpace(*in.Frequency) == "" {
		in.Frequency = nil
	}
	if err := validate(in.NeedKind, in.Frequency); err != nil {
		return CommunityNeed{}, err
	}
	if in.FoodID == 0 || in.Quantity.Sign() <= 0 {
		return CommunityNeed{}, errors.New("food_id and quantity are required")
	}
	if strings.TrimSpace(in.StartsOn) == "" {
		in.StartsOn = time.Now().Format("2006-01-02")
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	title := strings.TrimSpace(in.Title)
	var n CommunityNeed
	var ends sql.NullString
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO community_needs(food_id, title, quantity, need_kind, frequency, active, starts_on, ends_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, NULLIF($8, '')::date)
		RETURNING id, food_id, title, quantity, need_kind, frequency, active,
		          to_char(starts_on, 'YYYY-MM-DD'),
		          CASE WHEN ends_on IS NULL THEN NULL ELSE to_char(ends_on, 'YYYY-MM-DD') END,
		          created_at, updated_at`,
		in.FoodID, title, in.Quantity, in.NeedKind, in.Frequency, active, in.StartsOn, nullStr(in.EndsOn),
	).Scan(&n.ID, &n.FoodID, &n.Title, &n.Quantity, &n.NeedKind, &n.Frequency, &n.Active, &n.StartsOn, &ends, &n.CreatedAt, &n.UpdatedAt)
	if ends.Valid {
		n.EndsOn = &ends.String
	}
	return n, err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM community_needs WHERE id = $1`, id)
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

func nullStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func validate(kind string, freq *string) error {
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
			return errors.New("invalid frequency")
		}
	default:
		return errors.New("need_kind must be sporadic or recurring")
	}
}
