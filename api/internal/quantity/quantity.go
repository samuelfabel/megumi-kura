package quantity

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"strings"
)

// Q is a precise decimal quantity stored as NUMERIC in PostgreSQL.
// Values are kept as decimal strings to avoid float rounding.
type Q struct {
	rat *big.Rat
}

func New(s string) (Q, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Q{}, fmt.Errorf("empty quantity")
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return Q{}, fmt.Errorf("invalid quantity %q", s)
	}
	return Q{rat: r}, nil
}

func Must(s string) Q {
	q, err := New(s)
	if err != nil {
		panic(err)
	}
	return q
}

func FromInt(n int64) Q {
	return Q{rat: big.NewRat(n, 1)}
}

func Zero() Q {
	return Q{rat: big.NewRat(0, 1)}
}

func (q Q) String() string {
	if q.rat == nil {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(q.rat.FloatString(3), "0"), ".")
}

func (q Q) Raw() string {
	if q.rat == nil {
		return "0"
	}
	return q.rat.FloatString(3)
}

func (q Q) IsZero() bool {
	return q.rat == nil || q.rat.Sign() == 0
}

func (q Q) Sign() int {
	if q.rat == nil {
		return 0
	}
	return q.rat.Sign()
}

func (q Q) Less(other Q) bool {
	return q.rat.Cmp(other.ensure().rat) < 0
}

func (q Q) Cmp(other Q) int {
	return q.ensure().rat.Cmp(other.ensure().rat)
}

func (q Q) Add(other Q) Q {
	r := new(big.Rat).Add(q.ensure().rat, other.ensure().rat)
	return Q{rat: r}
}

func (q Q) Sub(other Q) Q {
	r := new(big.Rat).Sub(q.ensure().rat, other.ensure().rat)
	return Q{rat: r}
}

// DivFloorInt returns floor(q / other) as int64.
func (q Q) DivFloorInt(other Q) int64 {
	if other.IsZero() {
		return 0
	}
	quot := new(big.Rat).Quo(q.ensure().rat, other.ensure().rat)
	return new(big.Int).Div(quot.Num(), quot.Denom()).Int64()
}

func (q Q) ensure() Q {
	if q.rat == nil {
		return Zero()
	}
	return q
}

func (q Q) Value() (driver.Value, error) {
	return q.Raw(), nil
}

func (q *Q) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*q = Zero()
		return nil
	case []byte:
		parsed, err := New(string(v))
		if err != nil {
			return err
		}
		*q = parsed
		return nil
	case string:
		parsed, err := New(v)
		if err != nil {
			return err
		}
		*q = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into quantity", src)
	}
}

func (q Q) MarshalJSON() ([]byte, error) {
	return []byte(`"` + q.String() + `"`), nil
}

func (q *Q) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	parsed, err := New(s)
	if err != nil {
		return err
	}
	*q = parsed
	return nil
}
