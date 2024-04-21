package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"

	"kastelo.dev/sie"
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
		cw.Write([]string{acc.ID, acc.Type, acc.Description, acc.InBalance.String(), acc.OutBalance.String()})
	}
	cw.Flush()
	fd.Close()
}

func writeTransactions(dir string, doc *sie.Document) {
	totals := map[string]sie.Decimal{}
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
			if tot == 0 {
				for _, acc := range doc.Accounts {
					if acc.ID == tr.Account {
						tot += acc.InBalance
						break
					}
				}
				totals[tr.Account] = tot
			}
			tot += tr.Amount
			row := []string{
				en.ID,
				en.Date.Format("2006-01-02"),
				en.Type,
				en.Description,
				tr.Account,
				tr.Amount.String(),
				tot.String(),
			}
			cw.Write(row)
		}
	}
	cw.Flush()
	fd.Close()
}
