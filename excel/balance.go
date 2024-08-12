package excel

import (
	"time"

	"github.com/xuri/excelize/v2"
	"kastelo.dev/sie"
)

func BalanceXLSX(doc *sie.Document) ([]byte, error) {
	xlsx := excelize.NewFile()

	_ = xlsx.SetAppProps(&excelize.AppProperties{
		Application: "kastelo.dev/sie",
		Company:     "Kastelo AB",
		DocSecurity: 2,
	})

	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())

	// Set column widths
	_ = xlsx.SetColWidth(sheet, "A", "A", 8)
	_ = xlsx.SetColWidth(sheet, "B", "B", 50)
	_ = xlsx.SetColWidth(sheet, "C", "E", 15)

	writeBalanceSheet(xlsx, sheet, doc)
	_ = xlsx.SetSheetName(sheet, "Balansräkning")

	// Increase size of window
	for i := range xlsx.WorkBook.BookViews.WorkBookView {
		xlsx.WorkBook.BookViews.WorkBookView[i].XWindow = "1000"
		xlsx.WorkBook.BookViews.WorkBookView[i].YWindow = "1000"
		xlsx.WorkBook.BookViews.WorkBookView[i].WindowWidth = 25000
		xlsx.WorkBook.BookViews.WorkBookView[i].WindowHeight = 25000 / 3 * 2
	}

	buf, err := xlsx.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeBalanceSheet(xlsx *excelize.File, sheet string, doc *sie.Document) {
	state := 0
	var inSum, outSum sie.Decimal
	var assets, liabilities sie.Decimal
	row := 1
loop:
	for _, acc := range doc.Accounts {

		switch {
		case state == 0 && acc.ID >= 1000 && acc.ID <= 1999:
			style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thickBorder("top")))
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('F', row), style)
			row++

			_ = xlsx.SetCellValue(sheet, cell('B', row), "Tillgångar")
			_ = xlsx.SetCellValue(sheet, cell('C', row), "Ing balans")
			_ = xlsx.SetCellValue(sheet, cell('D', row), "Period")
			_ = xlsx.SetCellValue(sheet, cell('E', row), "Utg balans")
			style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thinBorder("bottom")))
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('B', row), style)
			style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thinBorder("bottom"), textAlignment("right")))
			_ = xlsx.SetCellStyle(sheet, cell('C', row), cell('E', row), style)
			row++

			state = 1

		case state == 1 && acc.ID >= 2000 && acc.ID <= 2999:
			_ = xlsx.SetCellValue(sheet, cell('A', row), "")
			_ = xlsx.SetCellValue(sheet, cell('B', row), "Summa tillgångar")
			_ = xlsx.SetCellValue(sheet, cell('C', row), inSum.Float64())
			_ = xlsx.SetCellValue(sheet, cell('D', row), (outSum - inSum).Float64())
			_ = xlsx.SetCellValue(sheet, cell('E', row), outSum.Float64())
			inSum = 0
			outSum = 0
			style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)
			row++
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)
			row++

			_ = xlsx.SetCellValue(sheet, cell('B', row), "Eget kapital, skulder")
			style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thinBorder("bottom")))
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('B', row), style)
			_ = xlsx.SetCellValue(sheet, cell('C', row), "Ing balans")
			_ = xlsx.SetCellValue(sheet, cell('D', row), "Period")
			_ = xlsx.SetCellValue(sheet, cell('E', row), "Utg balans")
			style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thinBorder("bottom"), textAlignment("right")))
			_ = xlsx.SetCellStyle(sheet, cell('C', row), cell('E', row), style)
			row++
			state = 2

		case acc.ID >= 3000 && acc.ID <= 3999:
			_ = xlsx.SetCellValue(sheet, cell('A', row), "")
			_ = xlsx.SetCellValue(sheet, cell('B', row), "Summa eget kapital, skulder")
			_ = xlsx.SetCellValue(sheet, cell('C', row), inSum.Float64())
			_ = xlsx.SetCellValue(sheet, cell('D', row), (outSum - inSum).Float64())
			_ = xlsx.SetCellValue(sheet, cell('E', row), outSum.Float64())
			inSum = 0
			outSum = 0
			style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)
			row++
			_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)
			row++
			break loop
		}

		if acc.InBalance == 0 && acc.OutBalance == 0 {
			continue
		}

		inSum += acc.InBalance
		outSum += acc.OutBalance

		switch state {
		case 1:
			assets += acc.OutBalance
		case 2:
			liabilities += acc.OutBalance
		}

		_ = xlsx.SetCellValue(sheet, cell('A', row), acc.ID)
		_ = xlsx.SetCellValue(sheet, cell('B', row), acc.Description)
		_ = xlsx.SetCellValue(sheet, cell('C', row), acc.InBalance.Float64())
		_ = xlsx.SetCellValue(sheet, cell('D', row), (acc.OutBalance - acc.InBalance).Float64())
		_ = xlsx.SetCellValue(sheet, cell('E', row), acc.OutBalance.Float64())
		style, _ := xlsx.NewStyle(mergeStyles(defaultStyle()))
		_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('B', row), style)
		style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), customNumberFormat()))
		_ = xlsx.SetCellStyle(sheet, cell('C', row), cell('E', row), style)

		row++
		_ = xlsx.SetCellStyle(sheet, cell('C', row), cell('E', row), style)
	}

	result := assets
	result += liabilities

	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thickBorder("bottom")))
	_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)
	row++

	_ = xlsx.SetCellValue(sheet, cell('A', row), "")
	_ = xlsx.SetCellValue(sheet, cell('B', row), "Beräknat resultat")
	_ = xlsx.SetCellValue(sheet, cell('E', row), result.Float64())
	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("bottom")))
	_ = xlsx.SetCellStyle(sheet, cell('A', row), cell('E', row), style)

	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle()))
	_ = xlsx.SetCellStyle(sheet, cell('F', 1), cell('F', row), style)
	_ = xlsx.SetCellStyle(sheet, cell('A', row+1), cell('F', row+1), style)
}

func balances(doc *sie.Document) map[int]*balance {
	balances := make(map[int]*balance)
	for _, acc := range doc.Accounts {
		balances[acc.ID] = newBalance()
		if acc.InBalance != 0 {
			balances[acc.ID].add(time.Time{}, acc.InBalance)
		}
	}
	for _, entry := range doc.Entries {
		for _, tran := range entry.Transactions {
			balances[tran.AccountID].add(entry.Date, tran.Amount)
		}
	}
	return balances
}

type balance struct {
	total  sie.Decimal
	months map[string][]sie.Decimal
}

func newBalance() *balance {
	return &balance{
		months: make(map[string][]sie.Decimal),
	}
}

func (b *balance) add(date time.Time, amount sie.Decimal) {
	b.total += amount
	key := date.Format("2006-01")
	b.months[key] = append(b.months[key], amount)
}

func (b *balance) inverse() *balance {
	new := newBalance()
	new.total -= b.total
	for m, v := range b.months {
		for i := range v {
			new.months[m] = append(new.months[m], -v[i])
		}
	}
	return new
}
