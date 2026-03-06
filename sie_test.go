package sie

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		in  string
		ok  bool
		out Decimal
	}{
		{"0", true, 0},
		{"0.00", true, 0},
		{"9.00", true, 900},
		{"9.50", true, 950},
		{"-9.50", true, -950},
		{"9.5", true, 950},
		{"-9.5", true, -950},
		{"banana", false, 0},
		{"1..2", false, 0},
	}

	for _, c := range cases {
		v, err := ParseDecimal(c.in)
		if c.ok && err != nil {
			t.Error("unexpected failure:", c.in)
		} else if !c.ok && err == nil {
			t.Error("unexpected success:", c.in)
		} else if v != c.out {
			t.Errorf("unexpected value %v != %v for %v", v, c.out, c.in)
		}
	}
}

func TestDocumentJSONRoundtrip(t *testing.T) {
	doc := Document{
		ProgramName:    "TestProgram",
		ProgramVersion: "1.0",
		GeneratedAt:    time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		GeneratedBy:    "tester",
		Type:           "E",
		OrgNo:          "556677-8899",
		CompanyName:    "Test AB",
		AccountPlan:    "BAS2024",
		Accounts: []Account{
			{ID: 1910, Type: "T", Description: "Kassa", InBalance: 150000, OutBalance: 175050},
			{ID: 3000, Type: "I", Description: "Intäkter", InBalance: 0, OutBalance: -250075},
		},
		Entries: []Entry{
			{
				ID:          "1",
				Type:        "V",
				Date:        time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
				Description: "Försäljning",
				Filed:       time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
				Transactions: []Transaction{
					{AccountID: 1910, Amount: 25050, Annotations: []Annotation{{Tag: 1, Text: "proj1", Description: "Projekt 1"}}},
					{AccountID: 3000, Amount: -25050},
				},
			},
		},
		Starts:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Ends:        time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Annotations: []Annotation{{Tag: 1, Text: "proj1", Description: "Projekt 1"}},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal("marshal:", err)
	}

	var got Document
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal("unmarshal:", err)
	}

	if !reflect.DeepEqual(doc, got) {
		t.Errorf("roundtrip mismatch\n got: %+v\nwant: %+v", got, doc)
	}
}

func TestDecimalJSONValues(t *testing.T) {
	cases := []struct {
		dec  Decimal
		json string
	}{
		{0, "0"},
		{100, "1"},
		{150, "1.50"},
		{-150, "-1.50"},
		{25050, "250.50"},
		{-25050, "-250.50"},
		{1000000, "10000"},
		{-1000000, "-10000"},
	}

	for _, c := range cases {
		data, err := json.Marshal(c.dec)
		if err != nil {
			t.Errorf("marshal %v: %v", c.dec, err)
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
		if got != c.dec {
			t.Errorf("unmarshal %s: got %v, want %v", data, got, c.dec)
		}
	}
}
