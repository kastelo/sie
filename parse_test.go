package sie

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(y int, m time.Month, d int) *timestamppb.Timestamp {
	return timestamppb.New(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

func TestParse(t *testing.T) {
	expected := &Document{
		ProgramName:        "SpeedLedger e-bokföring",
		ProgramVersion:     "2.0",
		GeneratedAt:        ts(2017, 3, 5),
		GeneratedBy:        "Jakob Borg",
		SieType:            SieType_SIE_TYPE_TRANSACTIONS,
		OrganisationNumber: "123456-7890",
		CompanyName:        "Kastelo AB",
		ChartOfAccounts:    "EUBAS97",
		FinancialYearStart: ts(2016, 1, 1),
		FinancialYearEnd:   ts(2016, 12, 31),
		Accounts: []*Account{
			{
				Number: 1930, Type: AccountType_ACCOUNT_TYPE_ASSET, Name: "Bankkonto",
				OpeningBalance: NewDecimal(0),
				ClosingBalance: NewDecimal(48043 * 100),
			},
			{
				Number: 2081, Type: AccountType_ACCOUNT_TYPE_LIABILITY, Name: "Aktiekapital",
				OpeningBalance: NewDecimal(0),
				ClosingBalance: NewDecimal(-50000 * 100),
			},
			{
				Number: 6310, Type: AccountType_ACCOUNT_TYPE_EXPENSE, Name: "Försäkringar",
				OpeningBalance: NewDecimal(0),
				ClosingBalance: NewDecimal(1957 * 100),
			},
		},
		Vouchers: []*Voucher{
			{
				Series:         "A",
				Number:         "1",
				Date:           ts(2016, 1, 2),
				RegisteredDate: ts(2016, 1, 3),
				Description:    "Aktiekapital",
				Postings: []*Posting{
					{AccountNumber: 1930, Dimensions: []*Dimension{{Number: 2, ObjectCode: "FOO"}}, Amount: NewDecimal(50000 * 100)},
					{AccountNumber: 2081, Dimensions: []*Dimension{{Number: 3, ObjectCode: "BAR"}}, Amount: NewDecimal(-50000 * 100)},
				},
			}, {
				Series:         "A",
				Number:         "2",
				Date:           ts(2016, 8, 29),
				RegisteredDate: ts(2016, 8, 30),
				Description:    "Försäkring F",
				Postings: []*Posting{
					{AccountNumber: 1930, Amount: NewDecimal(-1957 * 100)},
					{AccountNumber: 6310, Amount: NewDecimal(1957 * 100)},
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

func jsons(v any) string {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bs)
}
