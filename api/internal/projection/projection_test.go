package projection

import (
	"context"
	"database/sql"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func move(t *testing.T, db *sql.DB, foodID int64, typ, qty string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO stock_movements(food_id, movement_type, quantity) VALUES ($1, $2, $3)`,
		foodID, typ, qty); err != nil {
		t.Fatalf("insert movement: %v", err)
	}
}

func weeklyNeed(t *testing.T, db *sql.DB, foodID int64, qty, freq string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO community_needs(food_id, quantity, need_kind, frequency) VALUES ($1, $2, 'recurring', $3)`,
		foodID, qty, freq); err != nil {
		t.Fatalf("insert need: %v", err)
	}
}

func TestShortageRisksSeverity(t *testing.T) {
	db := testdb.Open(t)
	r := &Repository{DB: db}

	watch := testdb.Food(t, db, "arroz", "Arroz", "kg")
	critical := testdb.Food(t, db, "feijao", "Feijão", "kg")
	empty := testdb.Food(t, db, "oleo", "Óleo", "L")
	noNeed := testdb.Food(t, db, "sal", "Sal", "kg")

	// 10 kg for 7 kg/week -> 10 days -> watch
	move(t, db, watch, "IN", "10")
	weeklyNeed(t, db, watch, "7", "weekly")

	// 2 kg in stock + 1 kg promised for 7 kg/week -> 3 days -> critical
	move(t, db, critical, "IN", "2")
	weeklyNeed(t, db, critical, "7", "weekly")
	if _, err := db.Exec(`
		INSERT INTO promises(food_id, person_name, quantity, promise_date) VALUES ($1, 'Doador', 1, CURRENT_DATE)`,
		critical); err != nil {
		t.Fatalf("insert promise: %v", err)
	}

	// nothing available, monthly need -> critical with 0 days
	weeklyNeed(t, db, empty, "4", "monthly")

	move(t, db, noNeed, "IN", "1")

	risks, err := r.ShortageRisks(context.Background())
	if err != nil {
		t.Fatalf("shortage risks: %v", err)
	}
	byID := map[int64]ShortageRisk{}
	for _, risk := range risks {
		byID[risk.FoodID] = risk
	}

	check := func(id int64, severity string, alert bool, days *int) {
		t.Helper()
		got, ok := byID[id]
		if !ok {
			t.Fatalf("food %d missing from risks", id)
		}
		if got.Severity != severity || got.Alert != alert {
			t.Fatalf("food %s: severity=%s alert=%v, want %s %v", got.Code, got.Severity, got.Alert, severity, alert)
		}
		switch {
		case days == nil && got.DaysToEmpty != nil:
			t.Fatalf("food %s: days_to_empty=%d, want nil", got.Code, *got.DaysToEmpty)
		case days != nil && (got.DaysToEmpty == nil || *got.DaysToEmpty != *days):
			t.Fatalf("food %s: days_to_empty=%v, want %d", got.Code, got.DaysToEmpty, *days)
		}
	}
	ten, three, zero := 10, 3, 0
	check(watch, "watch", true, &ten)
	check(critical, "critical", true, &three)
	check(empty, "critical", true, &zero)
	check(noNeed, "ok", false, nil)

	if got := byID[critical].Promised; got.Cmp(quantity.Must("1")) != 0 {
		t.Fatalf("promised = %s, want 1", got)
	}
}

func TestTimeSeriesTodayTotals(t *testing.T) {
	db := testdb.Open(t)
	r := &Repository{DB: db}
	food := testdb.Food(t, db, "macarrao", "Macarrão", "kg")

	move(t, db, food, "IN", "5")
	move(t, db, food, "OUT", "2")
	if _, err := db.Exec(`
		INSERT INTO promises(food_id, person_name, quantity, promise_date) VALUES ($1, 'Doador', 3, CURRENT_DATE)`,
		food); err != nil {
		t.Fatalf("insert promise: %v", err)
	}

	var today string
	if err := db.QueryRow(`SELECT CURRENT_DATE::text`).Scan(&today); err != nil {
		t.Fatalf("current date: %v", err)
	}

	s, err := r.TimeSeries(context.Background(), food, 7)
	if err != nil {
		t.Fatalf("time series: %v", err)
	}
	if s.Code != "macarrao" || len(s.Points) == 0 {
		t.Fatalf("unexpected series: %+v", s)
	}
	last := s.Points[len(s.Points)-1]
	if last.Day != today {
		t.Fatalf("last point day = %s, want %s", last.Day, today)
	}
	if last.InQty.Cmp(quantity.Must("5")) != 0 || last.OutQty.Cmp(quantity.Must("2")) != 0 ||
		last.Promised.Cmp(quantity.Must("3")) != 0 || last.Balance.Cmp(quantity.Must("3")) != 0 {
		t.Fatalf("last point = %+v, want in=5 out=2 promised=3 balance=3", last)
	}

	if _, err := r.TimeSeries(context.Background(), 999999, 7); err == nil {
		t.Fatal("expected error for unknown food")
	}
}
