package sie // import "kastelo.io/sie"

import (
	"math/big"
	"time"
)

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
}

type Account struct {
	ID          string
	Type        string
	Description string
}

type Entry struct {
	ID           string
	Type         string
	Date         time.Time
	Description  string
	Transactions []Transaction
}

type Transaction struct {
	Account string
	Amount  *big.Rat
}
