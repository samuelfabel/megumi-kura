package stock

import (
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

func TestInsufficientStockError(t *testing.T) {
	if ErrInsufficientStock.Error() != "insufficient stock" {
		t.Fatal("unexpected message")
	}
	q := quantity.Must("10")
	if q.Less(quantity.Must("5")) {
		t.Fatal("10 should not be less than 5")
	}
	if !quantity.Must("4").Less(quantity.Must("5")) {
		t.Fatal("4 should be less than 5")
	}
}
