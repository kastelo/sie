package sie

import (
	"encoding/json"
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
		Starts:         time.Date(2016, 1, 2, 0, 0, 0, 0, time.UTC),
		Ends:           time.Date(2016, 8, 29, 0, 0, 0, 0, time.UTC),
		Accounts: []Account{
			{
				ID: "1930", Type: "T", Description: "Bankkonto",
				InBalance:  big.NewRat(0, 1),
				OutBalance: big.NewRat(48043, 1),
			},
			{
				ID: "2081", Type: "S", Description: "Aktiekapital",
				InBalance:  big.NewRat(0, 1),
				OutBalance: big.NewRat(-50000, 1),
			},
			{
				ID: "6310", Type: "K", Description: "Försäkringar",
				InBalance:  big.NewRat(0, 1),
				OutBalance: big.NewRat(1957, 1),
			},
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

	docStr := jsons(doc)
	expStr := jsons(expected)

	if docStr != expStr {
		t.Errorf("mismatch\n%s\n%s", docStr, expStr)
	}
}

func jsons(v interface{}) string {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bs)
}
