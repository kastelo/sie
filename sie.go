package sie

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Decimal int64 // "cents"

func (d Decimal) String() string {
	if d%100 == 0 {
		return d.FloatString(0)
	}
	return d.FloatString(2)
}

func (d Decimal) FloatString(decimals int) string {
	if decimals <= 0 {
		return fmt.Sprintf("%d", (d+50)/100)
	}
	abs := d
	if d < 0 {
		abs = -d
	}
	return fmt.Sprintf("%d.%0*d", d/100, decimals, abs%100)
}

func (d Decimal) Float64() float64 {
	return float64(d) / 100
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

func ParseDecimal(s string) (Decimal, error) {
	wholeStr, fracStr, ok := strings.Cut(s, ".")
	if !ok {
		wholeStr = s
		fracStr = "0"
	}
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse %q (whole part): %v", s, err)
	}
	frac, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse %q (fractional part): %v", s, err)
	}
	if whole < 0 {
		frac = -frac
	}
	return Decimal(whole*100 + frac), nil
}

type Document struct {
	ProgramName    string
	ProgramVersion string
	GeneratedAt    time.Time
	GeneratedBy    string
	Type           string
	OrgNo          string
	CompanyName    string
	AccountPlan    string
	Accounts       []Account
	Entries        []Entry
	Starts         time.Time
	Ends           time.Time
	Annotations    []Annotation
}

type Account struct {
	ID          int
	Type        string
	Description string
	InBalance   Decimal
	OutBalance  Decimal
}

type Entry struct {
	ID           string
	Type         string
	Date         time.Time
	Description  string
	Transactions []Transaction
}

type Transaction struct {
	AccountID   int
	Annotations []Annotation
	Amount      Decimal
}

type Annotation struct {
	Tag         int
	Text        string
	Description string
}

func (a Annotation) Equals(other Annotation) bool {
	return a.Tag == other.Tag && a.Text == other.Text
}

func (a Annotation) String() string {
	if a.Description != "" {
		return a.Description
	}
	return fmt.Sprintf("%d-%s", a.Tag, a.Text)
}

func (d *Document) CopyForAnnotation(ann Annotation) *Document {
	cpy := *d
	cpy.Entries = make([]Entry, 0, len(d.Entries))
	for _, e := range d.Entries {
		e2 := e
		e2.Transactions = make([]Transaction, 0, len(e.Transactions))
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
	return &cpy
}

func (d *Document) CopyWithoutAnnotations() *Document {
	cpy := *d
	cpy.Entries = make([]Entry, 0, len(d.Entries))
	for _, e := range d.Entries {
		e2 := e
		e2.Transactions = make([]Transaction, 0, len(e.Transactions))
		for _, t := range e.Transactions {
			if len(t.Annotations) == 0 {
				e2.Transactions = append(e2.Transactions, t)
			}
		}
		if len(e2.Transactions) > 0 {
			cpy.Entries = append(cpy.Entries, e2)
		}
	}
	return &cpy
}

func (d *Document) AddEntriesFrom(other *Document) {
	d.Entries = append(d.Entries, other.Entries...)
}
