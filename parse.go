package sie

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	iconv "github.com/djimenez/iconv-go"
)

func Parse(r io.Reader) (*Document, error) {
	convR, err := iconv.NewReader(r, "cp850", "utf-8")
	if err != nil {
		return nil, err
	}
	r = convR

	var doc Document
	var curVer Entry
	accountCache := make(map[string]int)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		words := splitWords(sc.Text())
		if len(words) < 1 {
			continue
		}

		switch words[0] {
		case "#FLAGGA":
			doc.Flag, _ = strconv.Atoi(words[1])

		case "#PROGRAM":
			doc.ProgramName = words[1]
			doc.ProgramVersion = words[2]

		case "#FORMAT":
			doc.Format = words[1]

		case "#GEN":
			if len(words) >= 2 {
				date, err := time.Parse("20060102", words[1])
				if err != nil {
					return nil, err
				}
				doc.GeneratedAt = date
			}
			if len(words) >= 3 {
				doc.GeneratedBy = words[2]
			}

		case "#SIETYP":
			doc.Type = words[1]

		case "#ORGNR":
			doc.OrgNo = words[1]

		case "#FNAMN":
			doc.CompanyName = words[1]

		case "#RAR":
			// not handled

		case "#KPTYP":
			doc.AccountPlan = words[1]

		case "#KONTO":
			acc := Account{
				ID:          words[1],
				Description: words[2],
			}
			accountCache[acc.ID] = len(doc.Accounts)
			doc.Accounts = append(doc.Accounts, acc)

		case "#KTYP":
			idx, ok := accountCache[words[1]]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[1])
			}
			doc.Accounts[idx].Type = words[2]

		case "#IB":
			if words[1] != "0" {
				continue
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			idx, ok := accountCache[words[2]]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[2])
			}
			doc.Accounts[idx].InBalance = amount

		case "#UB":
			if words[1] != "0" {
				continue
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			idx, ok := accountCache[words[2]]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[2])
			}
			doc.Accounts[idx].OutBalance = amount

		case "#VER":
			date, err := time.Parse("20060102", words[3])
			if err != nil {
				return nil, err
			}
			curVer = Entry{
				ID:          words[2],
				Type:        words[1],
				Date:        date,
				Description: words[4],
			}
			if doc.Starts.IsZero() || doc.Starts.After(date) {
				doc.Starts = date
			}
			if doc.Ends.IsZero() || doc.Ends.Before(date) {
				doc.Ends = date
			}

		case "#TRANS":
			var annotations []Annotation
			if words[2] != "" {
				// There's an annotation
				parts := strings.Split(words[2], " ")
				if len(parts)%2 != 0 {
					return nil, fmt.Errorf("annotation has odd number of parts")
				}
				for i := 0; i < len(parts); i += 2 {
					tagNo, _ := strconv.Atoi(maybeUnquote(parts[i]))
					text := maybeUnquote(parts[i+1])
					annotations = append(annotations, Annotation{Tag: tagNo, Text: text})
				}
			}
			amount, err := ParseDecimal(words[3])
			if err != nil {
				return nil, err
			}
			trans := Transaction{
				Account:     words[1],
				Amount:      amount,
				Annotations: annotations,
			}
			curVer.Transactions = append(curVer.Transactions, trans)

		case "}":
			doc.Entries = append(doc.Entries, curVer)
		}
	}

	sort.Slice(doc.Entries, func(i, j int) bool {
		return doc.Entries[i].Date.Before(doc.Entries[j].Date)
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
