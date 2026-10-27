package projection

import (
	"context"
	"database/sql"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

type Point struct {
	Day      string     `json:"day"` // YYYY-MM-DD
	Balance  quantity.Q `json:"balance"`
	InQty    quantity.Q `json:"in"`
	OutQty   quantity.Q `json:"out"`
	Promised quantity.Q `json:"promised"`
}

type Series struct {
	FoodID   int64   `json:"food_id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Points   []Point `json:"points"`
}

type ShortageRisk struct {
	FoodID       int64      `json:"food_id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Unit         string     `json:"unit"`
	Stock        quantity.Q `json:"stock"`
	Promised     quantity.Q `json:"promised"`
	ProjectedNeed quantity.Q `json:"projected_need"`
	DaysToEmpty  *int       `json:"days_to_empty,omitempty"`
	Alert        bool       `json:"alert"`
	Severity     string     `json:"severity"` // ok|watch|critical
}

type Repository struct {
	DB *sql.DB
}

// TimeSeries returns daily net stock balance and promise totals for a food over days.
func (r *Repository) TimeSeries(ctx context.Context, foodID int64, days int) (Series, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	var s Series
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, code, name, unit FROM foods WHERE id = $1 AND active = TRUE`, foodID,
	).Scan(&s.FoodID, &s.Code, &s.Name, &s.Unit)
	if err != nil {
		return Series{}, err
	}

	start := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	rows, err := r.DB.QueryContext(ctx, `
		WITH days AS (
			SELECT generate_series($2::date, CURRENT_DATE, '1 day'::interval)::date AS d
		),
		mov AS (
			SELECT created_at::date AS d,
				COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN quantity ELSE 0 END), 0) AS in_qty,
				COALESCE(SUM(CASE WHEN movement_type = 'OUT' THEN quantity ELSE 0 END), 0) AS out_qty
			FROM stock_movements
			WHERE food_id = $1 AND created_at::date >= $2::date
			GROUP BY created_at::date
		),
		prom AS (
			SELECT promise_date AS d, COALESCE(SUM(quantity), 0) AS qty
			FROM promises
			WHERE food_id = $1 AND promise_date >= $2::date
			GROUP BY promise_date
		),
		base AS (
			SELECT COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN quantity
			                         WHEN movement_type = 'OUT' THEN -quantity ELSE 0 END), 0) AS bal
			FROM stock_movements
			WHERE food_id = $1 AND created_at::date < $2::date
		)
		SELECT days.d::text,
			COALESCE(mov.in_qty, 0), COALESCE(mov.out_qty, 0), COALESCE(prom.qty, 0)
		FROM days
		LEFT JOIN mov ON mov.d = days.d
		LEFT JOIN prom ON prom.d = days.d
		ORDER BY days.d ASC`, foodID, start.Format("2006-01-02"))
	if err != nil {
		return Series{}, err
	}
	defer rows.Close()

	var base quantity.Q
	_ = r.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN quantity
		                         WHEN movement_type = 'OUT' THEN -quantity ELSE 0 END), 0)
		FROM stock_movements
		WHERE food_id = $1 AND created_at::date < $2::date`, foodID, start.Format("2006-01-02"),
	).Scan(&base)

	running := base
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Day, &p.InQty, &p.OutQty, &p.Promised); err != nil {
			return Series{}, err
		}
		running = running.Add(p.InQty).Sub(p.OutQty)
		p.Balance = running
		s.Points = append(s.Points, p)
	}
	return s, rows.Err()
}

// ShortageRisks estimates demand from recurring person needs + community needs
// versus stock+promised, flagging items likely to run short.
func (r *Repository) ShortageRisks(ctx context.Context) ([]ShortageRisk, error) {
	rows, err := r.DB.QueryContext(ctx, `
		WITH stock AS (
			SELECT f.id AS food_id, f.code, f.name, f.unit,
				COALESCE(SUM(CASE WHEN m.movement_type = 'IN' THEN m.quantity
				                  WHEN m.movement_type = 'OUT' THEN -m.quantity ELSE 0 END), 0) AS qty
			FROM foods f
			LEFT JOIN stock_movements m ON m.food_id = f.id
			WHERE f.active = TRUE
			GROUP BY f.id
		),
		promised AS (
			SELECT food_id, COALESCE(SUM(quantity), 0) AS qty
			FROM promises WHERE promise_date >= CURRENT_DATE
			GROUP BY food_id
		),
		need_weekly AS (
			SELECT food_id, COALESCE(SUM(
				CASE frequency
					WHEN 'weekly' THEN quantity
					WHEN 'biweekly' THEN quantity / 2
					WHEN 'monthly' THEN quantity / 4
					ELSE 0
				END
			), 0) AS weekly_qty
			FROM (
				SELECT food_id, quantity, frequency FROM person_needs
				WHERE active AND need_kind = 'recurring' AND frequency IS NOT NULL
				UNION ALL
				SELECT food_id, quantity, frequency FROM community_needs
				WHERE active AND need_kind = 'recurring' AND frequency IS NOT NULL
				  AND starts_on <= CURRENT_DATE AND (ends_on IS NULL OR ends_on >= CURRENT_DATE)
			) n
			GROUP BY food_id
		)
		SELECT s.food_id, s.code, s.name, s.unit, s.qty,
			COALESCE(p.qty, 0), COALESCE(w.weekly_qty, 0)
		FROM stock s
		LEFT JOIN promised p ON p.food_id = s.food_id
		LEFT JOIN need_weekly w ON w.food_id = s.food_id
		ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ShortageRisk
	for rows.Next() {
		var risk ShortageRisk
		var weekly quantity.Q
		if err := rows.Scan(&risk.FoodID, &risk.Code, &risk.Name, &risk.Unit,
			&risk.Stock, &risk.Promised, &weekly); err != nil {
			return nil, err
		}
		risk.ProjectedNeed = weekly
		available := risk.Stock.Add(risk.Promised)
		risk.Severity = "ok"
		if weekly.Sign() > 0 {
			// days ≈ available / (weekly/7) = available * 7 / weekly
			d := int(available.Mul(quantity.FromInt(7)).DivFloorInt(weekly))
			risk.DaysToEmpty = &d
			if d <= 7 {
				risk.Alert = true
				risk.Severity = "critical"
			} else if d <= 21 {
				risk.Alert = true
				risk.Severity = "watch"
			}
		}
		if available.Sign() <= 0 && weekly.Sign() > 0 {
			risk.Alert = true
			risk.Severity = "critical"
			zero := 0
			risk.DaysToEmpty = &zero
		}
		out = append(out, risk)
	}
	return out, rows.Err()
}
