package sie

import (
	"fmt"
	"math"
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

func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := string(b)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("unable to parse decimal %q: %v", s, err)
	}
	*d = Decimal(math.Round(f * 100))
	return nil
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

	if len(fracStr) == 1 {
		fracStr += "0"
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
	ProgramName    string       `json:"programName"`
	ProgramVersion string       `json:"programVersion"`
	GeneratedAt    time.Time    `json:"generatedAt"`
	GeneratedBy    string       `json:"generatedBy"`
	Type           string       `json:"type"`
	OrgNo          string       `json:"orgNo"`
	CompanyName    string       `json:"companyName"`
	AccountPlan    string       `json:"accountPlan"`
	Accounts       []Account    `json:"accounts"`
	Entries        []Entry      `json:"entries"`
	Starts         time.Time    `json:"starts"`
	Ends           time.Time    `json:"ends"`
	Annotations    []Annotation `json:"annotations"`
}

type Account struct {
	ID          int     `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	InBalance   Decimal `json:"inBalance"`
	OutBalance  Decimal `json:"outBalance"`
}

type Entry struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Date         time.Time     `json:"date"`
	Description  string        `json:"description"`
	Filed        time.Time     `json:"filed"`
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	AccountID   int          `json:"accountId"`
	Annotations []Annotation `json:"annotations"`
	Amount      Decimal      `json:"amount"`
}

type Annotation struct {
	Tag         int    `json:"tag"`
	Text        string `json:"text"`
	Description string `json:"description"`
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
