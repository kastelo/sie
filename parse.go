package sie

import (
	"bufio"
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Parse(r io.Reader) (*Document, error) {
	r = charmap.CodePage437.NewDecoder().Reader(r)

	var doc Document
	var curVoucher *Voucher
	accountCache := make(map[int]int)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		words := splitWords(sc.Text())
		if len(words) < 1 {
			continue
		}

		switch words[0] {
		case "#PROGRAM":
			doc.ProgramName = words[1]
			doc.ProgramVersion = words[2]

		case "#GEN":
			if len(words) >= 2 {
				date, err := time.Parse("20060102", words[1])
				if err != nil {
					return nil, err
				}
				doc.GeneratedAt = timestamppb.New(date)
			}
			if len(words) >= 3 {
				doc.GeneratedBy = words[2]
			}

		case "#SIETYP":
			doc.SieType = SieTypeFromCode(words[1])

		case "#ORGNR":
			doc.OrganisationNumber = words[1]

		case "#FNAMN":
			doc.CompanyName = words[1]

		case "#RAR":
			if words[1] == "0" {
				// Current financial year
				if t, err := time.Parse("20060102", words[2]); err == nil {
					doc.FinancialYearStart = timestamppb.New(t)
				}
				if t, err := time.Parse("20060102", words[3]); err == nil {
					doc.FinancialYearEnd = timestamppb.New(t)
				}
			}

		case "#KPTYP":
			doc.ChartOfAccounts = words[1]

		case "#KONTO":
			number := tryParseInt(words[1])
			acc := &Account{
				Number: int32(number),
				Name:   words[2],
			}
			accountCache[number] = len(doc.Accounts)
			doc.Accounts = append(doc.Accounts, acc)

		case "#KTYP":
			accID := tryParseInt(words[1])
			idx, ok := accountCache[accID]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[1])
			}
			doc.Accounts[idx].Type = AccountTypeFromCode(words[2])

		case "#IB":
			if words[1] != "0" {
				continue
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			accID := tryParseInt(words[2])
			idx, ok := accountCache[accID]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[2])
			}
			doc.Accounts[idx].OpeningBalance = amount

		case "#UB":
			if words[1] != "0" {
				continue
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			accID := tryParseInt(words[2])
			idx, ok := accountCache[accID]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[2])
			}
			doc.Accounts[idx].ClosingBalance = amount

		case "#VER":
			date, err := time.Parse("20060102", words[3])
			if err != nil {
				return nil, err
			}
			registered, err := time.Parse("20060102", words[5])
			if err != nil {
				return nil, err
			}
			curVoucher = &Voucher{
				Number:         words[2],
				Series:         words[1],
				Date:           timestamppb.New(date),
				Description:    words[4],
				RegisteredDate: timestamppb.New(registered),
			}
			if doc.FinancialYearStart == nil || doc.FinancialYearStart.AsTime().After(date) {
				doc.FinancialYearStart = timestamppb.New(date)
			}
			if doc.FinancialYearEnd == nil || doc.FinancialYearEnd.AsTime().Before(date) {
				doc.FinancialYearEnd = timestamppb.New(date)
			}

		case "#TRANS":
			var dimensions []*Dimension
			if words[2] != "" {
				// There's a dimension/object reference
				parts := strings.Split(words[2], " ")
				if len(parts)%2 != 0 {
					return nil, fmt.Errorf("dimension has odd number of parts")
				}
				for i := 0; i < len(parts); i += 2 {
					number, _ := strconv.Atoi(maybeUnquote(parts[i]))
					objectCode := maybeUnquote(parts[i+1])
					dimensions = append(dimensions, &Dimension{Number: int32(number), ObjectCode: objectCode})
				}
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			accID := tryParseInt(words[1])
			posting := &Posting{
				AccountNumber: int32(accID),
				Amount:        amount,
				Dimensions:    dimensions,
			}
			curVoucher.Postings = append(curVoucher.Postings, posting)

		case "#OBJEKT":
			number, _ := strconv.Atoi(words[1])
			objectCode := words[2]
			name := words[3]
			doc.Dimensions = append(doc.Dimensions, &Dimension{Number: int32(number), ObjectCode: objectCode, Name: name})

		case "}":
			var tot int64
			for _, p := range curVoucher.Postings {
				tot += p.Amount.GetCents()
			}
			if tot != 0 {
				return nil, fmt.Errorf("unbalanced voucher %v: %v", curVoucher, NewDecimal(tot).FloatString(2))
			}
			doc.Vouchers = append(doc.Vouchers, curVoucher)
		}
	}

	slices.SortFunc(doc.Accounts, func(a, b *Account) int {
		return cmp.Compare(a.Number, b.Number)
	})
	slices.SortFunc(doc.Vouchers, func(a, b *Voucher) int {
		if d := cmp.Compare(a.Date.AsTime().Unix(), b.Date.AsTime().Unix()); d != 0 {
			return d
		}
		return cmp.Compare(a.Number, b.Number)
	})
	slices.SortFunc(doc.Dimensions, func(a, b *Dimension) int {
		if d := cmp.Compare(a.Number, b.Number); d != 0 {
			return d
		}
		return cmp.Compare(a.Label(), b.Label())
	})

	return &doc, nil
}

func maybeUnquote(s string) string {
	if r, err := strconv.Unquote(s); err == nil {
		return r
	}
	if r, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return r
	}
	return s
}

func tryParseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
