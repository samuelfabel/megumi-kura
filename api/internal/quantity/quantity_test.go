package quantity

import "testing"

func TestAddSub(t *testing.T) {
	a := Must("20")
	b := Must("30")
	c := Must("10")
	got := a.Add(b).Sub(c)
	if got.String() != "40" {
		t.Fatalf("got %s want 40", got.String())
	}
}

func TestDivFloorInt(t *testing.T) {
	stock := Must("62")
	need := Must("5")
	if n := stock.DivFloorInt(need); n != 12 {
		t.Fatalf("got %d want 12", n)
	}
}

func TestNoFloatDrift(t *testing.T) {
	q := Must("0.1").Add(Must("0.2"))
	if q.String() != "0.3" {
		t.Fatalf("got %s want 0.3", q.String())
	}
}
