package sie

import (
	"reflect"
	"testing"
)

func TestSplitWords(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{
			"one two three",
			[]string{"one", "two", "three"},
		}, {
			`one "two three"`,
			[]string{"one", "two three"},
		}, {
			`one "two three" four`,
			[]string{"one", "two three", "four"},
		}, {
			`one "two three" "four"`,
			[]string{"one", "two three", "four"},
		}, {
			`one "two \" three" "four"`,
			[]string{"one", "two \" three", "four"},
		}, {
			`one "two \" three\\" "four"`,
			[]string{"one", "two \" three\\", "four"},
		}, {
			`one two {three \"four\"} "five"`,
			[]string{"one", "two", "three \"four\"", "five"},
		},
	}

	for _, tc := range cases {
		res := splitWords(tc.in)
		if !reflect.DeepEqual(res, tc.out) {
			t.Errorf("split(%q) -> %#v, expected %#v", tc.in, res, tc.out)
		}
	}
}
