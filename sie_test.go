package sie

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		in    string
		ok    bool
		cents int64
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
		} else if v.Cents != c.cents {
			t.Errorf("unexpected value %v != %v for %v", v, c.cents, c.in)
		}
	}
}

func TestDocumentJSONRoundtrip(t *testing.T) {
	doc := &Document{
		ProgramName:    "TestProgram",
		ProgramVersion: "1.0",
		GeneratedAt:    timestamppb.New(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)),
		GeneratedBy:    "tester",
		Type:           "E",
		OrgNo:          "556677-8899",
		CompanyName:    "Test AB",
		AccountPlan:    "BAS2024",
		Accounts: []*Account{
			{Id: 1910, Type: "T", Description: "Kassa", InBalance: &Decimal{Cents: 150000}, OutBalance: &Decimal{Cents: 175050}},
			{Id: 3000, Type: "I", Description: "Intäkter", InBalance: &Decimal{Cents: 0}, OutBalance: &Decimal{Cents: -250075}},
		},
		Entries: []*Entry{
			{
				Id:          "1",
				Type:        "V",
				Date:        timestamppb.New(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)),
				Description: "Försäljning",
				Filed:       timestamppb.New(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)),
				Transactions: []*Transaction{
					{AccountId: 1910, Amount: &Decimal{Cents: 25050}, Annotations: []*Annotation{{Tag: 1, Text: "proj1", Description: "Projekt 1"}}},
					{AccountId: 3000, Amount: &Decimal{Cents: -25050}},
				},
			},
		},
		Starts:      timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		Ends:        timestamppb.New(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
		Annotations: []*Annotation{{Tag: 1, Text: "proj1", Description: "Projekt 1"}},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal("marshal:", err)
	}

	var got Document
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal("unmarshal:", err)
	}

	if !reflect.DeepEqual(doc, &got) {
		t.Errorf("roundtrip mismatch\n got: %+v\nwant: %+v", &got, doc)
	}
}

func TestDecimalJSONValues(t *testing.T) {
	cases := []struct {
		dec  int64
		json string
	}{
		{0, "0.00"},
		{100, "1.00"},
		{150, "1.50"},
		{-150, "-1.50"},
		{25050, "250.50"},
		{-25050, "-250.50"},
		{1000000, "10000.00"},
		{-1000000, "-10000.00"},
	}

	for _, c := range cases {
		dec := &Decimal{Cents: c.dec}
		data, err := json.Marshal(dec)
		if err != nil {
			t.Errorf("marshal %v: %v", dec, err)
			continue
		}
		if string(data) != c.json {
			t.Errorf("marshal %v: got %s, want %s", c.dec, data, c.json)
		}

		var got Decimal
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal %s: %v", data, err)
			continue
		}
		if got.Cents != c.dec {
			t.Errorf("unmarshal %s: got %v, want %v", data, &got, c.dec)
		}
	}
}
