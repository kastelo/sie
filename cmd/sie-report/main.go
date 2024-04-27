package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"

	"kastelo.dev/sie"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lshortfile)

	cmdResult := kingpin.Command("result", "Show result report")
	cmdXLSX := kingpin.Command("xlsx", "Save result report as Excel")
	xlsxFile := cmdXLSX.Arg("file", "Output file name").Required().String()
	cmdBalance := kingpin.Command("balance", "Show balance report")
	cmdVAT := kingpin.Command("vat", "Show VAT report")
	infile := kingpin.Flag("input", "Input file").OpenFile(os.O_RDONLY, 0o666)
	cmd := kingpin.Parse()

	input := io.Reader(os.Stdin)
	if *infile != nil {
		input = *infile
	}

	switch cmd {
	case cmdResult.FullCommand():
		resultReport(balances(input))
	case cmdXLSX.FullCommand():
		doc, err := sie.Parse(input)
		if err != nil {
			log.Fatal(err)
		}
		bs, err := sie.ResultXLSX(doc)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*xlsxFile, bs, 0o666); err != nil {
			log.Fatal(err)
		}
	case cmdBalance.FullCommand():
		balanceReport(balances(input))
	case cmdVAT.FullCommand():
		vatReport(balances(input))
	}
}

func balances(r io.Reader) (*sie.Document, map[string]*balance) {
	doc, err := sie.Parse(r)
	if err != nil {
		log.Fatal(err)
	}

	balances := make(map[string]*balance)
	for _, acc := range doc.Accounts {
		balances[acc.ID] = newBalance()
		if acc.InBalance != 0 {
			balances[acc.ID].add(time.Time{}, acc.InBalance)
		}
	}
	for _, entry := range doc.Entries {
		for _, tran := range entry.Transactions {
			balances[tran.Account].add(entry.Date, tran.Amount)
		}
	}
	return doc, balances
}

func balanceReport(doc *sie.Document, accountBalance map[string]*balance) {
	state := 0
	var assets, liabilities sie.Decimal

loop:
	for _, acc := range doc.Accounts {
		switch {
		case state == 0 && strings.HasPrefix(acc.ID, "1"):
			fmt.Println("TILLGÅNGAR")
			state = 1

		case state == 1 && strings.HasPrefix(acc.ID, "2"):
			fmtAccount("", "Summa tillgångar", assets)
			fmt.Println("\nEGET KAPITAL, SKULDER")
			state = 2

		case strings.HasPrefix(acc.ID, "3"):
			fmtAccount("", "Summa eget kapital, skulder", liabilities)
			break loop
		}

		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}
		if bal.total == 0 {
			continue
		}

		switch state {
		case 1:
			assets += bal.total
		case 2:
			liabilities += bal.total
		}
		fmtAccount(acc.ID, acc.Description, bal.total)
	}

	fmt.Println("\nRESULTAT")
	result := assets
	result += liabilities
	fmtAccount("", "Beräknat resultat", result)
}

func vatReport(doc *sie.Document, accountBalance map[string]*balance) {
	var vat sie.Decimal

	fmt.Println("MOMS")
	for _, acc := range doc.Accounts {
		if !strings.HasPrefix(acc.ID, "26") {
			continue
		}

		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}

		vat += bal.total

		if bal.total != 0 {
			fmtAccount(acc.ID, acc.Description, bal.total)
		}
	}

	fmt.Println("\nSUMMA")
	fmtAccount("", "Moms att betala eller få tillbaka", vat)
}

func resultReport(doc *sie.Document, accountBalance map[string]*balance) {
	state := 0
	revenue := newBalance()
	extCost := newBalance()
	personnel := newBalance()
	financials := newBalance()

	for _, acc := range doc.Accounts {
		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}
		if bal.total == 0 {
			continue
		}
		bal = bal.inverse()

		switch {
		case state != 1 && strings.HasPrefix(acc.ID, "3"):
			headerMonths("OMSÄTTNING", doc.Starts, doc.Ends)
			dashes(doc.Starts, doc.Ends)
			state = 1

		case state != 2 && strings.HasPrefix(acc.ID, "4"):
			dashes(doc.Starts, doc.Ends)
			fmtAccountMonths("", "Nettoomsättning", doc.Starts, doc.Ends, revenue)
			fmt.Println()
			fmt.Println()
			headerMonths("EXTERNA KOSTNADER", doc.Starts, doc.Ends)
			dashes(doc.Starts, doc.Ends)
			state = 2

		case state != 3 && strings.HasPrefix(acc.ID, "7"):
			dashes(doc.Starts, doc.Ends)
			fmtAccountMonths("", "Summa externa konstnader", doc.Starts, doc.Ends, extCost)
			fmt.Println()
			fmt.Println()
			headerMonths("PERSONALKOSTNADER", doc.Starts, doc.Ends)
			dashes(doc.Starts, doc.Ends)
			state = 3

		case state != 4 && strings.HasPrefix(acc.ID, "8"):
			dashes(doc.Starts, doc.Ends)
			fmtAccountMonths("", "Summa personalkostnader", doc.Starts, doc.Ends, personnel)
			fmt.Println()
			fmt.Println()
			headerMonths("Finansiella poster", doc.Starts, doc.Ends)
			dashes(doc.Starts, doc.Ends)
			state = 4
		}

		switch state {
		case 0:
			continue
		case 1:
			revenue.addAll(bal)
		case 2:
			extCost.addAll(bal)
		case 3:
			personnel.addAll(bal)
		case 4:
			financials.addAll(bal)
		}

		fmtAccountMonths(acc.ID, acc.Description, doc.Starts, doc.Ends, bal)
	}
	dashes(doc.Starts, doc.Ends)
	if state == 3 {
		fmtAccountMonths("", "Summa personalkostnader", doc.Starts, doc.Ends, personnel)
	} else if state == 4 {
		fmtAccountMonths("", "Summa finansiella poster", doc.Starts, doc.Ends, financials)
	}
	fmt.Println()
	fmt.Println()
	headerMonths("RESULTAT", doc.Starts, doc.Ends)
	dashes(doc.Starts, doc.Ends)
	sum := revenue
	sum.addAll(extCost)
	sum.addAll(personnel)
	sum.addAll(financials)
	fmtAccountMonths("", "Resultat före skatt", doc.Starts, doc.Ends, sum)
}

func fmtAccount(id, descr string, val sie.Decimal) {
	const formatStr = "  %4s %-48s %10s\n"
	if len(descr) > 48 {
		descr = descr[:48]
	}
	fmt.Printf(formatStr, id, descr, val.String())
}

func fmtAccountMonths(id, descr string, starts, ends time.Time, bal *balance) {
	const formatStr = "  %4s %-48s"
	if len(descr) > 48 {
		descr = descr[:48]
	}
	fmt.Printf(formatStr, id, descr)
	t := starts
	for t.Before(ends) {
		val := "· "
		if v := bal.months[t.Format("2006-01")]; v != 0 {
			if str := v.FloatString(0); str != "0" {
				val = str
			}
		}
		fmt.Printf(" %8s", val)
		t = t.AddDate(0, 1, 0)
	}
	fmt.Printf(" | %8s", bal.total.FloatString(0))
	fmt.Printf("\n")
}

func headerMonths(hdr string, starts, ends time.Time) {
	fmt.Printf("%-55s", hdr)
	t := starts
	for t.Before(ends) {
		fmt.Printf(" %8s", t.Format("2006-01"))
		t = t.AddDate(0, 1, 0)
	}
	fmt.Printf(" | %-8s", "Total")
	fmt.Printf("\n")
}

func dashes(starts, ends time.Time) {
	fmt.Printf("%-55s", "")
	t := starts
	for t.Before(ends) {
		fmt.Printf("---------")
		t = t.AddDate(0, 1, 0)
	}
	fmt.Printf("-+---------")
	fmt.Printf("\n")
}
