package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
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
		doc, bal := balances(input)
		resultXLSX(*xlsxFile, doc, bal)
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

type section struct {
	name       string
	start, end int
}

var sections = []section{
	{"Nettoomsättning", 3000, 3799},
	{"Övriga rörelseintäkter", 3800, 3999},
	{"Varukostnader", 4000, 4999},
	{"Externa kostnader", 5000, 6999},
	{"Personalkostnader", 7000, 7699},
	{"Övrigt & finansiellt", 7700, 8999},
}

func resultXLSX(dst string, doc *sie.Document, accountBalance map[string]*balance) {
	sec := -1
	row := 1
	startRow := 1
	var sumRows []int
	xlsx := excelize.NewFile()

	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	xlsx.SetColWidth(sheet, "B", "B", 55)
	xlsx.SetColWidth(sheet, "C", "K", 10)

	sy, sm, _ := doc.Starts.Date()
	ey, em, _ := doc.Ends.Date()
	numMonths := (ey-sy)*12 + int(em) - int(sm) + 1

	style, _ := xlsx.NewStyle(styleJSON(defaultStyle()))
	xlsx.SetCellStyle(sheet, cell('A', 1), cell('A'+rune(numMonths)+5, 1000), style)

	xlsxHeaderMonths(xlsx, row, "", doc.Starts, doc.Ends)
	row++

	for _, acc := range doc.Accounts {
		bal, ok := accountBalance[acc.ID]
		if !ok {
			continue
		}
		if bal.total == 0 {
			continue
		}
		bal = bal.inverse()

		newSec := -1
		for i, sec := range sections {
			id := acc.IDInt()
			if sec.start <= id && id <= sec.end {
				newSec = i
				break
			}
		}

		if newSec != sec {
			if sec != -1 {
				xlsxSumMonths(xlsx, row, "", doc.Starts, doc.Ends, startRow)
				sumRows = append(sumRows, row)
				row++
			}

			row++
			xlsxHeader(xlsx, row, numMonths, sections[newSec].name)
			row++
			startRow = row
			sec = newSec
		}

		if newSec == -1 {
			continue
		}

		xlsxAccountMonths(xlsx, row, acc.ID, acc.Description, doc.Starts, doc.Ends, bal)
		row++
	}

	xlsxSumMonths(xlsx, row, "", doc.Starts, doc.Ends, startRow)
	sumRows = append(sumRows, row)
	row++
	row++
	xlsxSumSumMonths(xlsx, row, doc.Starts, doc.Ends, sumRows)
	row++
	row++

	style, _ = xlsx.NewStyle("")
	xlsx.SetCellStyle(sheet, cell('A', row+2), cell('A'+rune(numMonths)+5, 1000), style)

	xlsx.SaveAs(dst)
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

func cell(col rune, row int) string {
	return fmt.Sprintf("%c%d", col, row)
}

func xlsxAccountMonths(xlsx *excelize.File, row int, id, descr string, starts, ends time.Time, bal *balance) {
	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	idInt, _ := strconv.Atoi(id)
	xlsx.SetCellInt(sheet, cell('A', row), idInt)
	xlsx.SetCellValue(sheet, cell('B', row), descr)
	t := starts
	col := 'C'
	for t.Before(ends) {
		if v := bal.months[t.Format("2006-01")]; v != 0 {
			xlsx.SetCellValue(sheet, cell(col, row), v.Float64())
		}
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(C%d:%c%d)", row, col-1, row))
	style, _ := xlsx.NewStyle(styleJSON(defaultStyle(), customNumberFormat()))
	xlsx.SetCellStyle(sheet, cell('C', row), cell(col, row), style)
	style, _ = xlsx.NewStyle(styleJSON(defaultStyle(), fontItalic(), customNumberFormat()))
	xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
}

func defaultStyle() map[string]any {
	return map[string]any{
		// solid white
		"fill": map[string]any{
			"type":    "pattern",
			"color":   []any{"#FFFFFF"},
			"pattern": 1,
		},
	}
}

func customNumberFormat() map[string]any {
	return map[string]any{"custom_number_format": "#,##0,"}
}

func fontItalic() map[string]any {
	return map[string]any{"font": map[string]any{"italic": true}}
}

func fontBold() map[string]any {
	return map[string]any{"font": map[string]any{"bold": true}}
}

func fontBoldItalic() map[string]any {
	return map[string]any{"font": map[string]any{"bold": true, "italic": true}}
}

func textAlignment(a string) map[string]any {
	return map[string]any{"alignment": map[string]any{"horizontal": a}}
}

func thinBorder(where ...string) map[string]any {
	var borders []any
	for _, w := range where {
		borders = append(borders, map[string]any{
			"type":  w,
			"color": "#000000",
			"style": 1,
		})
	}
	return map[string]any{
		"border": borders,
	}
}

func thickBorder(where ...string) map[string]any {
	var borders []any
	for _, w := range where {
		borders = append(borders, map[string]any{
			"type":  w,
			"color": "#000000",
			"style": 2,
		})
	}
	return map[string]any{
		"border": borders,
	}
}

func styleJSON(ext ...map[string]any) string {
	m := map[string]any{}
	for _, e := range ext {
		maps.Copy(m, e)
	}
	bs, _ := json.Marshal(m)
	return string(bs)
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

func xlsxHeaderMonths(xlsx *excelize.File, row int, hdr string, starts, ends time.Time) {
	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	xlsx.SetCellValue(sheet, cell('B', row), hdr)
	t := starts
	col := 'C'
	for t.Before(ends) {
		xlsx.SetCellValue(sheet, cell(col, row), t.Format("2006-01"))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	xlsx.SetCellValue(sheet, cell(col, row), "Total")

	style, _ := xlsx.NewStyle(styleJSON(defaultStyle(), fontBold(), textAlignment("right")))
	xlsx.SetCellStyle(sheet, cell('B', row), cell(col, row), style)
}

func xlsxHeader(xlsx *excelize.File, row, cols int, hdr string) {
	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	xlsx.SetCellValue(sheet, cell('B', row), hdr)
	style, _ := xlsx.NewStyle(styleJSON(defaultStyle(), fontBold(), thinBorder("bottom")))
	xlsx.SetCellStyle(sheet, cell('B', row), cell('B'+rune(cols)+2, row), style)
}

func xlsxSumMonths(xlsx *excelize.File, row int, hdr string, starts, ends time.Time, startRow int) {
	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	xlsx.SetCellValue(sheet, cell('B', row), hdr)
	t := starts
	col := 'C'
	for t.Before(ends) {
		xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(%c%d:%c%d)", col, startRow, col, row-1))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(%c%d:%c%d)", col, startRow, col, row-1))

	style, _ := xlsx.NewStyle(styleJSON(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
	xlsx.SetCellStyle(sheet, cell('B', row), cell(col-1, row), style)

	style, _ = xlsx.NewStyle(styleJSON(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("top")))
	xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
}

func sumcells(col rune, rows []int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%c%d", col, rows[0])
	for _, row := range rows[1:] {
		fmt.Fprintf(&b, "+%c%d", col, row)
	}
	return b.String()
}

func xlsxSumSumMonths(xlsx *excelize.File, row int, starts, ends time.Time, sumRows []int) {
	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	xlsx.SetCellValue(sheet, cell('B', row), "Resultat")

	// sum

	t := starts
	col := 'C'
	for t.Before(ends) {
		xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))

	// accumulated sum

	row++

	xlsx.SetCellValue(sheet, cell('B', row), "Ackumulerat resultat")

	col = 'C'
	xlsx.SetCellFormula(sheet, cell(col, row), cell(col, row-1))

	t = starts.AddDate(0, 1, 0)
	col = 'D'
	for t.Before(ends) {
		xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("%c%d+%c%d", col-1, row, col, row-1))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	style, _ := xlsx.NewStyle(styleJSON(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
	xlsx.SetCellStyle(sheet, cell('B', row-1), cell(col-1, row-1), style)

	style, _ = xlsx.NewStyle(styleJSON(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("top")))
	xlsx.SetCellStyle(sheet, cell(col, row-1), cell(col, row-1), style)

	style, _ = xlsx.NewStyle(styleJSON(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("bottom")))
	xlsx.SetCellStyle(sheet, cell('B', row), cell(col-1, row), style)

	style, _ = xlsx.NewStyle(styleJSON(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("bottom")))
	xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
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
