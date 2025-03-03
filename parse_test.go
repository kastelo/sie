package sie

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"
)

func TestParse(t *testing.T) {
	expected := &Document{
		ProgramName:    "SpeedLedger e-bokföring",
		ProgramVersion: "2.0",
		GeneratedAt:    time.Date(2017, 3, 5, 0, 0, 0, 0, time.UTC),
		GeneratedBy:    "Jakob Borg",
		Type:           "4",
		OrgNo:          "123456-7890",
		CompanyName:    "Kastelo AB",
		AccountPlan:    "EUBAS97",
		Starts:         time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC),
		Ends:           time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC),
		Accounts: []Account{
			{
				ID: 1930, Type: "T", Description: "Bankkonto",
				InBalance:  0,
				OutBalance: 48043 * 100,
			},
			{
				ID: 2081, Type: "S", Description: "Aktiekapital",
				InBalance:  0,
				OutBalance: -50000 * 100,
			},
			{
				ID: 6310, Type: "K", Description: "Försäkringar",
				InBalance:  0,
				OutBalance: 1957 * 100,
			},
		},
		Entries: []Entry{
			{
				Type:        "A",
				ID:          "1",
				Date:        time.Date(2016, 1, 2, 0, 0, 0, 0, time.UTC),
				Filed:       time.Date(2016, 1, 3, 0, 0, 0, 0, time.UTC),
				Description: "Aktiekapital",
				Transactions: []Transaction{
					{AccountID: 1930, Annotations: []Annotation{{Tag: 2, Text: "FOO"}}, Amount: 50000 * 100},
					{AccountID: 2081, Annotations: []Annotation{{Tag: 3, Text: "BAR"}}, Amount: -50000 * 100},
				},
			}, {
				Type:        "A",
				ID:          "2",
				Date:        time.Date(2016, 8, 29, 0, 0, 0, 0, time.UTC),
				Filed:       time.Date(2016, 8, 30, 0, 0, 0, 0, time.UTC),
				Description: "Försäkring F",
				Transactions: []Transaction{
					{AccountID: 1930, Amount: -1957 * 100},
					{AccountID: 6310, Amount: 1957 * 100},
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
		t.Error(godiffpatch.GeneratePatch("rendered", string(expStr), string(docStr)))
	}
}

func jsons(v interface{}) string {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bs)
}
