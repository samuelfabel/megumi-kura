package kit

import (
	"context"
	"errors"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
	"github.com/samuelfabel/megumi-kura/api/internal/testdb"
)

func newRepo(t *testing.T) (*Repository, *stock.Repository) {
	t.Helper()
	db := testdb.Open(t)
	st := &stock.Repository{DB: db}
	return &Repository{DB: db, Stock: st}, st
}

func moveIn(t *testing.T, st *stock.Repository, foodID int64, qty string) {
	t.Helper()
	if _, err := st.MoveIn(context.Background(), foodID, quantity.Must(qty), nil, nil); err != nil {
		t.Fatalf("move in: %v", err)
	}
}

func TestCreateRequiresCodeAndName(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	if _, err := r.Create(ctx, CreateInput{Code: "  ", Name: "Kit"}); err == nil {
		t.Fatal("expected error for blank code")
	}
	if _, err := r.Create(ctx, CreateInput{Code: "kit", Name: ""}); err == nil {
		t.Fatal("expected error for blank name")
	}
	k, err := r.Create(ctx, CreateInput{Code: " kit_idosos ", Name: " Kit idosos "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if k.Code != "kit_idosos" || k.Name != "Kit idosos" || !k.Active {
		t.Fatalf("unexpected kit: %+v", k)
	}
}

func TestAvailabilityUsesLimitingItem(t *testing.T) {
	r, st := newRepo(t)
	ctx := context.Background()
	rice := testdb.Food(t, r.DB, "arroz", "Arroz", "kg")
	beans := testdb.Food(t, r.DB, "feijao", "Feijão", "kg")

	k, err := r.Create(ctx, CreateInput{Code: "kit_teste", Name: "Kit teste"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	k, err = r.ReplaceItems(ctx, k.ID, []PutItem{
		{FoodID: rice, Quantity: quantity.Must("5")},
		{FoodID: beans, Quantity: quantity.Must("2")},
	})
	if err != nil {
		t.Fatalf("replace items: %v", err)
	}
	if k.Available != 0 {
		t.Fatalf("available without stock = %d, want 0", k.Available)
	}

	moveIn(t, st, rice, "12")  // 2 kits
	moveIn(t, st, beans, "10") // 5 kits

	k, err = r.Get(ctx, k.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if k.Available != 2 {
		t.Fatalf("available = %d, want 2", k.Available)
	}
	if k.LimitingCode != "arroz" {
		t.Fatalf("limiting = %q, want arroz", k.LimitingCode)
	}
}

func TestReplaceItemsValidation(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	food := testdb.Food(t, r.DB, "oleo", "Óleo", "L")
	k, err := r.Create(ctx, CreateInput{Code: "kit_v", Name: "Kit V"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.ReplaceItems(ctx, k.ID, nil); err == nil {
		t.Fatal("expected error for empty item list")
	}
	if _, err := r.ReplaceItems(ctx, k.ID, []PutItem{{FoodID: food, Quantity: quantity.Zero()}}); err == nil {
		t.Fatal("expected error for zero quantity")
	}
	if _, err := r.ReplaceItems(ctx, 999999, []PutItem{{FoodID: food, Quantity: quantity.Must("1")}}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing kit err = %v, want ErrNotFound", err)
	}
}

func TestDefaultKitSyncsLegacyBasket(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	sugar := testdb.Food(t, r.DB, "acucar", "Açúcar", "kg")

	var kitID int64
	if err := r.DB.QueryRow(`SELECT id FROM kits WHERE code = 'cesta_basica'`).Scan(&kitID); err != nil {
		t.Fatalf("default kit from migration: %v", err)
	}
	if _, err := r.ReplaceItems(ctx, kitID, []PutItem{{FoodID: sugar, Quantity: quantity.Must("1.5")}}); err != nil {
		t.Fatalf("replace items: %v", err)
	}

	var n int
	var qty quantity.Q
	if err := r.DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(quantity), 0) FROM basket_items WHERE food_id = $1`, sugar).Scan(&n, &qty); err != nil {
		t.Fatalf("query basket_items: %v", err)
	}
	if n != 1 || qty.Cmp(quantity.Must("1.5")) != 0 {
		t.Fatalf("basket_items = %d rows, qty %s; want 1 row, 1.5", n, qty)
	}
}

func TestExpandItemsAndListActiveOnly(t *testing.T) {
	r, _ := newRepo(t)
	ctx := context.Background()
	milk := testdb.Food(t, r.DB, "leite", "Leite", "L")

	k, err := r.Create(ctx, CreateInput{Code: "kit_leite", Name: "Kit leite"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := r.ReplaceItems(ctx, k.ID, []PutItem{{FoodID: milk, Quantity: quantity.Must("3")}}); err != nil {
		t.Fatalf("replace items: %v", err)
	}

	items, err := r.ExpandItems(ctx, k.ID)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(items) != 1 || items[0].FoodID != milk || items[0].Quantity.Cmp(quantity.Must("3")) != 0 {
		t.Fatalf("expanded = %+v", items)
	}

	inactive := false
	if _, err := r.Update(ctx, k.ID, CreateInput{Code: k.Code, Name: k.Name}, &inactive); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	active, err := r.List(ctx, true)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, a := range active {
		if a.ID == k.ID {
			t.Fatal("inactive kit returned by List(activeOnly=true)")
		}
	}
	if _, err := r.Update(ctx, 999999, CreateInput{Code: "x", Name: "x"}, &inactive); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing kit err = %v, want ErrNotFound", err)
	}
}
