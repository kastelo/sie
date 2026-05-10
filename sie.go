package sie

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
)

func (d *Decimal) FloatString(decimals int) string {
	if decimals <= 0 {
		r := d.GetCents() / 100
		if d.GetCents()%100 >= 50 {
			r++
		} else if d.GetCents()%100 <= -50 {
			r--
		}
		return fmt.Sprintf("%d", r)
	}
	abs := d.GetCents()
	if abs < 0 {
		abs = -abs
	}
	return fmt.Sprintf("%d.%0*d", d.GetCents()/100, decimals, abs%100)
}

func (d *Decimal) Float64() float64 {
	return float64(d.GetCents()) / 100
}

func (d *Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.FloatString(2)), nil
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := string(b)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("unable to parse decimal %q: %v", s, err)
	}
	d.Cents = int64(math.Round(f * 100))
	return nil
}

func (d *Decimal) Set(o *Decimal) {
	d.Cents = o.GetCents()
}

func (d *Decimal) Copy() *Decimal {
	return &Decimal{Cents: d.GetCents()}
}

func (d *Decimal) Adding(o *Decimal) *Decimal {
	return &Decimal{Cents: d.GetCents() + o.GetCents()}
}

func (d *Decimal) Subtracting(o *Decimal) *Decimal {
	return &Decimal{Cents: d.GetCents() - o.GetCents()}
}

func (d *Decimal) Inverse() *Decimal {
	return &Decimal{Cents: -d.GetCents()}
}

func ParseDecimal(s string) (*Decimal, error) {
	neg := strings.HasPrefix(s, "-")

	wholeStr, fracStr, ok := strings.Cut(s, ".")
	if !ok {
		wholeStr = s
		fracStr = "0"
	}
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return &Decimal{}, fmt.Errorf("unable to parse %q (whole part): %v", s, err)
	}

	// Normalize fractional part to exactly 2 digits (truncate >2, pad <2),
	// then round based on the third digit if present.
	round := int64(0)
	if len(fracStr) > 2 {
		if fracStr[2] >= '5' {
			round = 1
		}
		fracStr = fracStr[:2]
	}
	if len(fracStr) == 1 {
		fracStr += "0"
	}
	frac, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return &Decimal{}, fmt.Errorf("unable to parse %q (fractional part): %v", s, err)
	}
	frac += round

	if neg {
		frac = -frac
	}
	return &Decimal{Cents: whole*100 + frac}, nil
}

func (a *Annotation) Equals(other *Annotation) bool {
	return a.Tag == other.Tag && a.Text == other.Text
}

func (d *Document) CopyForAnnotation(ann *Annotation) *Document {
	cpy := proto.Clone(d).(*Document)
	cpy.Entries = make([]*Entry, 0, len(d.Entries))
	for _, e := range d.Entries {
		e2 := e
		e2.Transactions = make([]*Transaction, 0, len(e.Transactions))
		for _, t := range e.Transactions {
			for _, a := range t.Annotations {
				if a.Equals(ann) {
					e2.Transactions = append(e2.Transactions, t)
					break
				}
			}
		}
		if len(e2.Transactions) > 0 {
			cpy.Entries = append(cpy.Entries, e2)
		}
	}
	return cpy
}

func (d *Document) CopyWithoutAnnotations() *Document {
	cpy := proto.Clone(d).(*Document)
	cpy.Entries = make([]*Entry, 0, len(d.Entries))
	for _, e := range d.Entries {
		e2 := e
		e2.Transactions = make([]*Transaction, 0, len(e.Transactions))
		for _, t := range e.Transactions {
			if len(t.Annotations) == 0 {
				e2.Transactions = append(e2.Transactions, t)
			}
		}
		if len(e2.Transactions) > 0 {
			cpy.Entries = append(cpy.Entries, e2)
		}
	}
	return cpy
}

func (d *Document) AddEntriesFrom(other *Document) {
	d.Entries = append(d.Entries, other.Entries...)
}
