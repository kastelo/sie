package sie

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
)

// The Document, Account, Voucher, Posting, Dimension and Decimal types are
// generated from proto/sie.proto (see sie.pb.go). This file adds the
// convenience methods on top of the generated types.

// Decimal represents a monetary amount as an integer number of hundredths
// ("cents"). A nil *Decimal is treated as zero throughout, which lets it be
// used as a zero-valued accumulator.

// NewDecimal returns a Decimal holding the given number of cents.
func NewDecimal(cents int64) *Decimal {
	return &Decimal{Cents: cents}
}

// Float64 returns the amount as a float64. This is lossy for large values and
// is intended for display and reporting, not for arithmetic on the amounts.
// GetCents is the generated nil-safe getter, so a nil Decimal reads as zero.
func (d *Decimal) Float64() float64 {
	return float64(d.GetCents()) / 100
}

// Add returns the sum of d and o as a new Decimal.
func (d *Decimal) Add(o *Decimal) *Decimal {
	return &Decimal{Cents: d.GetCents() + o.GetCents()}
}

// Sub returns d minus o as a new Decimal.
func (d *Decimal) Sub(o *Decimal) *Decimal {
	return &Decimal{Cents: d.GetCents() - o.GetCents()}
}

// Neg returns the negation of d as a new Decimal.
func (d *Decimal) Neg() *Decimal {
	return &Decimal{Cents: -d.GetCents()}
}

// IsZero reports whether d represents zero (including a nil Decimal).
func (d *Decimal) IsZero() bool {
	return d.GetCents() == 0
}

// Sign returns -1, 0 or +1 depending on the sign of d.
func (d *Decimal) Sign() int {
	switch c := d.GetCents(); {
	case c < 0:
		return -1
	case c > 0:
		return 1
	default:
		return 0
	}
}

// FloatString formats the amount with the given number of decimals. With zero
// (or fewer) decimals it rounds to the nearest whole unit.
func (d *Decimal) FloatString(decimals int) string {
	c := d.GetCents()
	if decimals <= 0 {
		r := c / 100
		if c%100 >= 50 {
			r++
		} else if c%100 <= -50 {
			r--
		}
		return fmt.Sprintf("%d", r)
	}
	abs := c
	if c < 0 {
		abs = -c
	}
	return fmt.Sprintf("%d.%0*d", c/100, decimals, abs%100)
}

func ParseDecimal(s string) (*Decimal, error) {
	neg := strings.HasPrefix(s, "-")

	wholeStr, fracStr, ok := strings.Cut(s, ".")
	if !ok {
		wholeStr = s
		fracStr = "0"
	}
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %q (whole part): %v", s, err)
	}

	// Normalize fractional part to exactly 2 digits (truncate >2, pad <2),
	// then round based on the third digit if present.
	round := int64(0)
	if len(fracStr) > 2 {
		if fracStr[2] >= '5' {
			round = 1
		}
		fracStr = fracStr[:2]
	}
	if len(fracStr) == 1 {
		fracStr += "0"
	}
	frac, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %q (fractional part): %v", s, err)
	}
	frac += round

	if neg {
		frac = -frac
	}
	return &Decimal{Cents: whole*100 + frac}, nil
}

// AccountTypeFromCode maps a SIE #KTYP code ("T"/"S"/"K"/"I") to an
// AccountType, returning ACCOUNT_TYPE_UNSPECIFIED for anything else.
func AccountTypeFromCode(code string) AccountType {
	switch code {
	case "T":
		return AccountType_ACCOUNT_TYPE_ASSET
	case "S":
		return AccountType_ACCOUNT_TYPE_LIABILITY
	case "K":
		return AccountType_ACCOUNT_TYPE_EXPENSE
	case "I":
		return AccountType_ACCOUNT_TYPE_INCOME
	default:
		return AccountType_ACCOUNT_TYPE_UNSPECIFIED
	}
}

// Code returns the SIE #KTYP code for the account type, or the empty string
// when unspecified.
func (t AccountType) Code() string {
	switch t {
	case AccountType_ACCOUNT_TYPE_ASSET:
		return "T"
	case AccountType_ACCOUNT_TYPE_LIABILITY:
		return "S"
	case AccountType_ACCOUNT_TYPE_EXPENSE:
		return "K"
	case AccountType_ACCOUNT_TYPE_INCOME:
		return "I"
	default:
		return ""
	}
}

// SieTypeFromCode maps a SIE #SIETYP code ("1".."4") to a SieType, returning
// SIE_TYPE_UNSPECIFIED for anything else. The enum values match the codes.
func SieTypeFromCode(code string) SieType {
	n, err := strconv.Atoi(code)
	if err != nil {
		return SieType_SIE_TYPE_UNSPECIFIED
	}
	if _, ok := SieType_name[int32(n)]; ok && n != 0 {
		return SieType(n)
	}
	return SieType_SIE_TYPE_UNSPECIFIED
}

// Code returns the SIE #SIETYP code for the type, or the empty string when
// unspecified.
func (t SieType) Code() string {
	if t == SieType_SIE_TYPE_UNSPECIFIED {
		return ""
	}
	return strconv.Itoa(int(t))
}

// Equals reports whether two dimensions refer to the same dimension number and
// object code. The name is deliberately ignored, matching the identity used
// when filtering postings.
func (a *Dimension) Equals(other *Dimension) bool {
	return a.GetNumber() == other.GetNumber() && a.GetObjectCode() == other.GetObjectCode()
}

// Label returns a human-readable name for the dimension, preferring the object
// name when present. The generated String method renders the proto text form
// instead, so we can't reuse that name.
func (a *Dimension) Label() string {
	if a.GetName() != "" {
		return a.GetName()
	}
	return fmt.Sprintf("%d-%s", a.GetNumber(), a.GetObjectCode())
}

// CopyForDimension returns a copy of the document containing only the postings
// carrying the given dimension, dropping any vouchers left empty.
func (d *Document) CopyForDimension(dim *Dimension) *Document {
	cpy := proto.Clone(d).(*Document)
	cpy.Vouchers = nil
	for _, v := range d.Vouchers {
		var postings []*Posting
		for _, p := range v.Postings {
			for _, dm := range p.Dimensions {
				if dm.Equals(dim) {
					postings = append(postings, p)
					break
				}
			}
		}
		if len(postings) > 0 {
			v2 := proto.Clone(v).(*Voucher)
			v2.Postings = postings
			cpy.Vouchers = append(cpy.Vouchers, v2)
		}
	}
	return cpy
}

// CopyWithoutDimensions returns a copy of the document containing only the
// postings that carry no dimensions, dropping any vouchers left empty.
func (d *Document) CopyWithoutDimensions() *Document {
	cpy := proto.Clone(d).(*Document)
	cpy.Vouchers = nil
	for _, v := range d.Vouchers {
		var postings []*Posting
		for _, p := range v.Postings {
			if len(p.Dimensions) == 0 {
				postings = append(postings, p)
			}
		}
		if len(postings) > 0 {
			v2 := proto.Clone(v).(*Voucher)
			v2.Postings = postings
			cpy.Vouchers = append(cpy.Vouchers, v2)
		}
	}
	return cpy
}

func (d *Document) AddVouchersFrom(other *Document) {
	d.Vouchers = append(d.Vouchers, other.Vouchers...)
}
