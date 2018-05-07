package main

import (
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/kastelo/sie"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lshortfile)

	cmdResult := kingpin.Command("result", "Show result report")
	cmdBalance := kingpin.Command("balance", "Show balance report")
	cmdVAT := kingpin.Command("vat", "Show VAT report")
	infile := kingpin.Flag("input", "Input file").OpenFile(os.O_RDONLY, 0666)
	cmd := kingpin.Parse()

	input := io.Reader(os.Stdin)
	if *infile != nil {
		input = *infile
	}

	switch cmd {
	case cmdResult.FullCommand():
		resultReport(balances(input))
	case cmdBalance.FullCommand():
		balanceReport(balances(input))
	case cmdVAT.FullCommand():
		vatReport(balances(input))
	}
}

func balances(r io.Reader) (*sie.Document, map[string]*big.Rat) {
	doc, err := sie.Parse(r)
	if err != nil {
		log.Fatal(err)
	}

	balances := make(map[string]*big.Rat)
	for _, acc := range doc.Accounts {
		bal := &big.Rat{}
		if acc.InBalance != nil {
			bal.Add(bal, acc.InBalance)
			balances[acc.ID] = bal
		}
	}
	for _, entry := range doc.Entries {
		for _, tran := range entry.Transactions {
			bal, ok := balances[tran.Account]
			if !ok {
				bal = &big.Rat{}
				balances[tran.Account] = bal
			}
			bal.Add(bal, tran.Amount)
		}
	}
	return doc, balances
}

func balanceReport(doc *sie.Document, accountBalance map[string]*big.Rat) {
	state := 0
	var assets, liabilities big.Rat

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
		if bal.Num().Int64() == 0 {
			continue
		}

		switch state {
		case 1:
			assets.Add(&assets, bal)
		case 2:
			liabilities.Add(&liabilities, bal)
		}
		fmtAccount(acc.ID, acc.Description, *bal)
	}

	fmt.Println("\nRESULTAT")
	result := assets
	result.Add(&result, &liabilities)
	fmtAccount("", "Beräknat resultat", result)
}

func vatReport(doc *sie.Document, accountBalance map[string]*big.Rat) {
	var vat big.Rat

	fmt.Println("MOMS")
	for _, acc := range doc.Accounts {
		if !strings.HasPrefix(acc.ID, "26") {
			continue
		}

		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}

		vat.Add(&vat, bal)

		fmtAccount(acc.ID, acc.Description, *bal)
	}

	fmt.Println("\nSUMMA")
	fmtAccount("", "Moms att betala eller få tillbaka", vat)
}

func resultReport(doc *sie.Document, accountBalance map[string]*big.Rat) {
	state := 0
	var revenue, extCost, personnel, financials big.Rat

	for _, acc := range doc.Accounts {
		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}
		bal.Sub(&big.Rat{}, bal)
		if bal.Num().Int64() == 0 {
			continue
		}

		switch {
		case state != 1 && strings.HasPrefix(acc.ID, "3"):
			fmt.Println("OMSÄTTNING")
			state = 1

		case state != 2 && strings.HasPrefix(acc.ID, "5"):
			fmtAccount("", "Nettoomsättning", revenue)
			fmt.Println("\nEXTERNA KOSTNADER")
			state = 2

		case state != 3 && strings.HasPrefix(acc.ID, "7"):
			fmtAccount("", "Summa externa konstnader", extCost)
			fmt.Println("\nPERSONALKOSTNADER")
			state = 3

		case state != 4 && strings.HasPrefix(acc.ID, "8"):
			fmtAccount("", "Summa personalkostnader", personnel)
			fmt.Println("\nFinansiella poster")
			state = 4
		}

		switch state {
		case 0:
			continue
		case 1:
			revenue.Add(&revenue, bal)
		case 2:
			extCost.Add(&extCost, bal)
		case 3:
			personnel.Add(&personnel, bal)
		case 4:
			financials.Add(&financials, bal)
		}

		fmtAccount(acc.ID, acc.Description, *bal)
	}
	fmtAccount("", "Summa finansiella poster", financials)
	fmt.Println("\nRESULTAT")
	sum := revenue
	sum.Add(&sum, &extCost)
	sum.Add(&sum, &personnel)
	sum.Add(&sum, &financials)
	fmtAccount("", "Resultat före skatt", sum)
}

func fmtAccount(id, descr string, val big.Rat) {
	const formatStr = "  %4s %-48s %10s\n"
	fmt.Printf(formatStr, id, descr, val.FloatString(2))
}
