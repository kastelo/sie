package main

import (
	"encoding/csv"
	"flag"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"

	"kastelo.io/int/sie"
)

func main() {
	dir := flag.String("dir", ".", "Directory")
	flag.Parse()

	doc, err := sie.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	writeAccounts(*dir, doc)
	writeTransactions(*dir, doc)
}

func writeAccounts(dir string, doc *sie.Document) {
	fd, err := os.Create(filepath.Join(dir, "accounts.csv"))
	if err != nil {
		log.Fatal(err)
	}
	cw := csv.NewWriter(fd)
	cw.Write([]string{"AccountID", "Type", "Description", "InBalance", "OutBalance"})
	for _, acc := range doc.Accounts {
		cw.Write([]string{acc.ID, acc.Type, acc.Description, formatRat(acc.InBalance), formatRat(acc.OutBalance)})
	}
	cw.Flush()
	fd.Close()
}

func writeTransactions(dir string, doc *sie.Document) {
	totals := map[string]*big.Rat{}
	fd, err := os.Create(filepath.Join(dir, "transactions.csv"))
	if err != nil {
		log.Fatal(err)
	}
	cw := csv.NewWriter(fd)
	cw.Write([]string{"TransactionID", "Date", "Type", "Description", "AccountID", "Amount", "Total"})
	sort.Slice(doc.Entries, func(a, b int) bool {
		ae := doc.Entries[a]
		be := doc.Entries[b]
		if ae.Date.Equal(be.Date) {
			return ae.ID < be.ID
		}
		return ae.Date.Before(be.Date)
	})
	for _, en := range doc.Entries {
		for _, tr := range en.Transactions {
			tot := totals[tr.Account]
			if tot == nil {
				tot = new(big.Rat)
				for _, acc := range doc.Accounts {
					if acc.ID == tr.Account {
						if acc.InBalance != nil {
							tot.Add(tot, acc.InBalance)
						}
						break
					}
				}
				totals[tr.Account] = tot
			}
			tot.Add(tot, tr.Amount)
			row := []string{
				en.ID,
				en.Date.Format("2006-01-02"),
				en.Type,
				en.Description,
				tr.Account,
				formatRat(tr.Amount),
				formatRat(tot),
			}
			cw.Write(row)
		}
	}
	cw.Flush()
	fd.Close()
}

func formatRat(rat *big.Rat) string {
	if rat == nil {
		return "0.00"
	}
	return rat.FloatString(2)
}
