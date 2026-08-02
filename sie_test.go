package sie

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		in  string
		ok  bool
		out int64 // cents
	}{
		{"0", true, 0},
		{"0.00", true, 0},
		{"9.00", true, 900},
		{"9.50", true, 950},
		{"-9.50", true, -950},
		{"9.5", true, 950},
		{"-9.5", true, -950},
		{"-0.50", true, -50},
		{"-0.5", true, -50},
		{"1.125", true, 113},
		{"-1.125", true, -113},
		{"1.999", true, 200},
		{"1.1", true, 110},
		{"1.100", true, 110},
		{"banana", false, 0},
		{"1..2", false, 0},
	}

	for _, c := range cases {
		v, err := ParseDecimal(c.in)
		if c.ok && err != nil {
			t.Error("unexpected failure:", c.in)
		} else if !c.ok && err == nil {
			t.Error("unexpected success:", c.in)
		} else if err == nil && v.GetCents() != c.out {
			t.Errorf("unexpected value %v != %v for %v", v.GetCents(), c.out, c.in)
		}
	}
}

func TestDocumentProtoRoundtrip(t *testing.T) {
	doc := &Document{
		ProgramName:        "TestProgram",
		ProgramVersion:     "1.0",
		GeneratedAt:        timestamppb.New(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)),
		GeneratedBy:        "tester",
		SieType:            SieType_SIE_TYPE_TRANSACTIONS,
		OrganisationNumber: "556677-8899",
		CompanyName:        "Test AB",
		ChartOfAccounts:    "BAS2024",
		Accounts: []*Account{
			{Number: 1910, Type: AccountType_ACCOUNT_TYPE_ASSET, Name: "Kassa", OpeningBalance: NewDecimal(150000), ClosingBalance: NewDecimal(175050)},
			{Number: 3000, Type: AccountType_ACCOUNT_TYPE_INCOME, Name: "Intäkter", OpeningBalance: NewDecimal(0), ClosingBalance: NewDecimal(-250075)},
		},
		Vouchers: []*Voucher{
			{
				Number:         "1",
				Series:         "V",
				Date:           timestamppb.New(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)),
				Description:    "Försäljning",
				RegisteredDate: timestamppb.New(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)),
				Postings: []*Posting{
					{AccountNumber: 1910, Amount: NewDecimal(25050), Dimensions: []*Dimension{{Number: 1, ObjectCode: "proj1", Name: "Projekt 1"}}},
					{AccountNumber: 3000, Amount: NewDecimal(-25050)},
				},
			},
		},
		FinancialYearStart: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		FinancialYearEnd:   timestamppb.New(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
		Dimensions:         []*Dimension{{Number: 1, ObjectCode: "proj1", Name: "Projekt 1"}},
	}

	data, err := proto.Marshal(doc)
	if err != nil {
		t.Fatal("marshal:", err)
	}

	var got Document
	if err := proto.Unmarshal(data, &got); err != nil {
		t.Fatal("unmarshal:", err)
	}

	if !proto.Equal(doc, &got) {
		t.Errorf("roundtrip mismatch\n got: %v\nwant: %v", &got, doc)
	}
}

func TestDecimalFormat(t *testing.T) {
	cases := []struct {
		cents    int64
		float    float64
		floatStr string // FloatString(2)
	}{
		{0, 0, "0.00"},
		{100, 1, "1.00"},
		{150, 1.5, "1.50"},
		{-150, -1.5, "-1.50"},
		{25050, 250.5, "250.50"},
		{-25050, -250.5, "-250.50"},
		{1000000, 10000, "10000.00"},
		{-1000000, -10000, "-10000.00"},
	}

	for _, c := range cases {
		d := NewDecimal(c.cents)
		if got := d.Float64(); got != c.float {
			t.Errorf("Float64(%d) = %v, want %v", c.cents, got, c.float)
		}
		if got := d.FloatString(2); got != c.floatStr {
			t.Errorf("FloatString(%d) = %q, want %q", c.cents, got, c.floatStr)
		}
	}
}
