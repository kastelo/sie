package sie

import (
	"time"
)

type balance struct {
	total  Decimal
	months map[string]Decimal
}

func newBalance() *balance {
	return &balance{
		months: make(map[string]Decimal),
	}
}

func (b *balance) add(date time.Time, amount Decimal) {
	b.total += amount
	key := date.Format("2006-01")
	b.months[key] += amount
}

func (b *balance) inverse() *balance {
	new := newBalance()
	new.total -= b.total
	for k, v := range b.months {
		new.months[k] = -v
	}
	return new
}

func (b *balance) addAll(other *balance) {
	b.total += other.total
	for key, v := range other.months {
		b.months[key] += v
	}
}
