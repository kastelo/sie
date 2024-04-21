package sie

import (
	"fmt"
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
		return fmt.Sprintf("%d", d/100)
	}
	return fmt.Sprintf("%d.%0*d", d/100, decimals, d%100)
}

func (d Decimal) Float64() float64 {
	return float64(d) / 100
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
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
