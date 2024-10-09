package sie

import "testing"

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
