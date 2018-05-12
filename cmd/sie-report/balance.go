package main

import (
	"math/big"
	"time"
)

type balance struct {
	total  *big.Rat
	months map[string]*big.Rat
}

func newBalance() *balance {
	return &balance{
		total:  &big.Rat{},
		months: make(map[string]*big.Rat),
	}
}

func (b *balance) add(date time.Time, amount *big.Rat) {
	b.total.Add(b.total, amount)
	key := date.Format("2006-01")
	if _, ok := b.months[key]; !ok {
		b.months[key] = &big.Rat{}
	}
	b.months[key].Add(b.months[key], amount)
}

func (b *balance) inverse() *balance {
	new := newBalance()
	new.total.Sub(new.total, b.total)
	for k, v := range b.months {
		var b big.Rat
		b.Sub(&b, v)
		new.months[k] = &b
	}
	return new
}

func (b *balance) addAll(other *balance) {
	b.total.Add(b.total, other.total)
	for key, v := range other.months {
		if _, ok := b.months[key]; !ok {
			b.months[key] = &big.Rat{}
		}
		b.months[key].Add(b.months[key], v)
	}
}
