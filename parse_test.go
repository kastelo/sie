package sie

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	expected := &Document{
		Flag:           0,
		ProgramName:    "SpeedLedger e-bokföring",
		ProgramVersion: "2.0",
		Format:         "PC8",
		GeneratedAt:    time.Date(2017, 3, 5, 0, 0, 0, 0, time.UTC),
		GeneratedBy:    "Jakob Borg",
		Type:           "4",
		OrgNo:          "123456-7890",
		CompanyName:    "Kastelo AB",
		AccountPlan:    "EUBAS97",
		Accounts: []Account{
			{ID: "1110", Type: "T", Description: "Byggnader"},
			{ID: "1119", Type: "T", Description: "Ack avskrivn byggnader"},
		},
		Entries: []Entry{
			{
				Type:        "A",
				ID:          "1",
				Date:        time.Date(2016, 1, 2, 0, 0, 0, 0, time.UTC),
				Description: "Aktiekapital",
				Transactions: []Transaction{
					{Account: "1930", Amount: big.NewRat(50000, 1)},
					{Account: "2081", Amount: big.NewRat(-50000, 1)},
				},
			}, {
				Type:        "A",
				ID:          "2",
				Date:        time.Date(2016, 8, 29, 0, 0, 0, 0, time.UTC),
				Description: "Försäkring F",
				Transactions: []Transaction{
					{Account: "1930", Amount: big.NewRat(-1957, 1)},
					{Account: "6310", Amount: big.NewRat(1957, 1)},
				},
			},
		},
	}

	fd, _ := os.Open("testdata/testdata.se")
	doc, err := Parse(fd)
	if err != nil {
		t.Fatal(err)
	}

	docStr := fmt.Sprintf("%+v", doc)
	expStr := fmt.Sprintf("%+v", expected)

	if docStr != expStr {
		t.Errorf("mismatch\n%s\n%s", docStr, expStr)
	}
}
