package sie

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParse(t *testing.T) {
	expected := &Document{
		ProgramName:    "SpeedLedger e-bokföring",
		ProgramVersion: "2.0",
		GeneratedAt:    timestamppb.New(time.Date(2017, 3, 5, 0, 0, 0, 0, time.UTC)),
		GeneratedBy:    "Jakob Borg",
		Type:           "4",
		OrgNo:          "123456-7890",
		CompanyName:    "Kastelo AB",
		AccountPlan:    "EUBAS97",
		Starts:         timestamppb.New(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Ends:           timestamppb.New(time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC)),
		Accounts: []*Account{
			{
				Id: 1930, Type: "T", Description: "Bankkonto",
				InBalance:  &Decimal{Cents: 0},
				OutBalance: &Decimal{Cents: 48043 * 100},
			},
			{
				Id: 2081, Type: "S", Description: "Aktiekapital",
				InBalance:  &Decimal{Cents: 0},
				OutBalance: &Decimal{Cents: -50000 * 100},
			},
			{
				Id: 6310, Type: "K", Description: "Försäkringar",
				InBalance:  &Decimal{Cents: 0},
				OutBalance: &Decimal{Cents: 1957 * 100},
			},
		},
		Entries: []*Entry{
			{
				Type:        "A",
				Id:          "1",
				Date:        timestamppb.New(time.Date(2016, 1, 2, 0, 0, 0, 0, time.UTC)),
				Filed:       timestamppb.New(time.Date(2016, 1, 3, 0, 0, 0, 0, time.UTC)),
				Description: "Aktiekapital",
				Transactions: []*Transaction{
					{AccountId: 1930, Annotations: []*Annotation{{Tag: 2, Text: "FOO"}}, Amount: &Decimal{Cents: 50000 * 100}},
					{AccountId: 2081, Annotations: []*Annotation{{Tag: 3, Text: "BAR"}}, Amount: &Decimal{Cents: -50000 * 100}},
				},
			}, {
				Type:        "A",
				Id:          "2",
				Date:        timestamppb.New(time.Date(2016, 8, 29, 0, 0, 0, 0, time.UTC)),
				Filed:       timestamppb.New(time.Date(2016, 8, 30, 0, 0, 0, 0, time.UTC)),
				Description: "Försäkring F",
				Transactions: []*Transaction{
					{AccountId: 1930, Amount: &Decimal{Cents: -1957 * 100}},
					{AccountId: 6310, Amount: &Decimal{Cents: 1957 * 100}},
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
