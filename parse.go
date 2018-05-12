package sie

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"time"

	iconv "github.com/djimenez/iconv-go"
)

func Parse(r io.Reader) (*Document, error) {
	if convR, err := iconv.NewReader(r, "cp850", "utf-8"); err == nil {
		r = convR
	}

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
			date, err := time.Parse("20060102", words[1])
			if err != nil {
				return nil, err
			}
			doc.GeneratedAt = date
			doc.GeneratedBy = words[2]

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
			amount, ok := new(big.Rat).SetString(words[3])
			if !ok {
				return nil, fmt.Errorf("unable to parse %s", words[3])
			}
			idx, ok := accountCache[words[2]]
			if !ok {
				return nil, fmt.Errorf("unknown account %q", words[2])
			}
			doc.Accounts[idx].InBalance = amount

		case "#UB":
			amount, ok := new(big.Rat).SetString(words[3])
			if !ok {
				return nil, fmt.Errorf("unable to parse %s", words[3])
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
			amount, ok := new(big.Rat).SetString(words[3])
			if !ok {
				return nil, fmt.Errorf("unable to parse %s", words[3])
			}
			trans := Transaction{
				Account: words[1],
				Amount:  amount,
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
