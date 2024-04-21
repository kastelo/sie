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
	Flag           int
	ProgramName    string
	ProgramVersion string
	Format         string
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
}

type Account struct {
	ID          string
	Type        string
	Description string
	InBalance   Decimal
	OutBalance  Decimal
}

func (a *Account) IDInt() int {
	id, _ := strconv.Atoi(a.ID[:4])
	return id
}

type Entry struct {
	ID           string
	Type         string
	Date         time.Time
	Description  string
	Transactions []Transaction
}

type Transaction struct {
	Account     string
	Annotations []Annotation
	Amount      Decimal
}

type Annotation struct {
	Tag  int
	Text string
}
