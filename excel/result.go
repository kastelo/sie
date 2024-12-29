package excel

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/xuri/excelize/v2"
	"kastelo.dev/sie"
)

type section struct {
	name       string
	start, end int
}

var sections = []section{
	{"Nettoomsättning", 3000, 3799},
	{"Aktiverat arbete för egen räkning", 3800, 3899},
	{"Övriga rörelseintäkter", 3900, 3999},
	{"Varukostnader", 4000, 4999},
	{"Externa kostnader", 5000, 6999},
	{"Personalkostnader", 7000, 7699},
	{"Av- och nedskrivningar", 7700, 7899},
	{"Övriga rörelsekostnader", 7900, 7999},
	{"Finansiella poster", 8000, 8998},
}

type summary struct {
	name        string
	sectionIdxs []int
	afterIdx    int
}

var summaries = []summary{
	{"Rörelsens intäkter", []int{0, 1, 2}, 2},
	{"Rörelsens kostnader", []int{3, 4, 5}, 5},
	{"Rörelseresultat", []int{0, 1, 2, 3, 4, 5}, 5},
}

func ResultXLSX(doc *sie.Document) ([]byte, error) {
	xlsx := excelize.NewFile()

	_ = xlsx.SetAppProps(&excelize.AppProperties{
		Application: "kastelo.dev/sie",
		Company:     "Kastelo AB",
		DocSecurity: 2,
	})

	sheet := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	writeSheet(xlsx, sheet, doc)
	_ = xlsx.SetSheetName(sheet, "Totalt")

	// For each annotation, create a new sheet

	type annotatedDoc struct {
		name string
		doc  *sie.Document
	}

	var docs []annotatedDoc
	for _, annotation := range doc.Annotations {
		doc.CopyForAnnotation(annotation)
		if len(doc.Entries) == 0 {
			continue
		}

		name := annotation.String()
		found := false
		for i := range docs {
			if docs[i].name == name {
				docs[i].doc.AddEntriesFrom(doc.CopyForAnnotation(annotation))
				found = true
				break
			}
		}
		if !found {
			docs = append(docs, annotatedDoc{name, doc.CopyForAnnotation(annotation)})
		}
	}

	for _, adoc := range docs {
		_, err := xlsx.NewSheet(adoc.name)
		if err != nil {
			return nil, err
		}
		writeSheet(xlsx, adoc.name, adoc.doc)
	}

	// If there were annotations, also produce a sheet for whatever remains

	if len(doc.Annotations) > 0 {
		cpy := doc.CopyWithoutAnnotations()
		_, _ = xlsx.NewSheet("(Other)")
		writeSheet(xlsx, "(Other)", cpy)
	}

	xlsx.SetActiveSheet(0)

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

func writeSheet(xlsx *excelize.File, sheet string, doc *sie.Document) {
	sec := -1
	row := 1
	startRow := 1
	var sumRows []int

	_ = xlsx.SetColWidth(sheet, "B", "B", 55)
	_ = xlsx.SetColWidth(sheet, "C", "K", 10)

	sy, sm, _ := doc.Starts.Date()
	ey, em, _ := doc.Ends.Date()
	numMonths := (ey-sy)*12 + int(em) - int(sm) + 1

	// Eget kapital vid årets ingång
	var inCapital sie.Decimal
	for _, acc := range doc.Accounts {
		if acc.ID == 2081 || acc.ID == 2091 {
			inCapital -= acc.InBalance
		}
	}

	style, _ := xlsx.NewStyle(defaultStyle())
	_ = xlsx.SetCellStyle(sheet, cell('A', 1), cell('A'+rune(numMonths)+5, 1000), style)

	xlsxHeaderMonths(xlsx, sheet, row, "", doc.Starts, doc.Ends)
	row++

	_ = xlsx.SetPanes(sheet, &excelize.Panes{
		ActivePane:  "bottomRight",
		Freeze:      true,
		XSplit:      2,
		YSplit:      1,
		TopLeftCell: "C2",
	})

	accountBalance := balances(doc)
	summarySumRows := make(map[string][]int)
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
			id := acc.ID
			if sec.start <= id && id <= sec.end {
				newSec = i
				break
			}
		}

		if newSec == -1 {
			continue
		}

		if newSec != sec {
			if sec != -1 {
				for _, sum := range summaries {
					if slices.Contains(sum.sectionIdxs, sec) {
						summarySumRows[sum.name] = append(summarySumRows[sum.name], row)
					}
				}

				xlsxSumMonths(xlsx, sheet, row, "", doc.Starts, doc.Ends, startRow)
				sumRows = append(sumRows, row)
				row++

				for _, sum := range summaries {
					if sum.afterIdx == sec {
						row++
						xlsxSectionSum(xlsx, sheet, row, sum.name, doc.Starts, doc.Ends, summarySumRows[sum.name])
						row++
					}
				}
			}

			row++
			xlsxHeader(xlsx, sheet, row, numMonths, sections[newSec].name)
			row++
			startRow = row
			sec = newSec
		}

		if newSec == -1 {
			continue
		}

		xlsxAccountMonths(xlsx, sheet, row, acc.ID, acc.Description, doc.Starts, doc.Ends, bal)
		row++
	}

	xlsxSumMonths(xlsx, sheet, row, "", doc.Starts, doc.Ends, startRow)
	sumRows = append(sumRows, row)
	row++
	row++
	xlsxSumSumMonths(xlsx, sheet, row, doc.Starts, doc.Ends, sumRows, accountBalance, inCapital)
	row++
	row++

	style, _ = xlsx.NewStyle(nil)
	_ = xlsx.SetCellStyle(sheet, cell('A', row+5), cell('A'+rune(numMonths)+5, 1000), style)
}

func cell(col rune, row int) string {
	return fmt.Sprintf("%c%d", col, row)
}

func xlsxAccountMonths(xlsx *excelize.File, sheet string, row int, id int, descr string, starts, ends time.Time, bal *balance) {
	_ = xlsx.SetCellInt(sheet, cell('A', row), id)
	_ = xlsx.SetCellValue(sheet, cell('B', row), descr)
	t := starts
	col := 'C'
	for !t.After(ends) {
		if v := bal.months[t.Format("2006-01")]; len(v) == 1 {
			_ = xlsx.SetCellValue(sheet, cell(col, row), v[0].amount.Float64())
		} else if len(v) != 0 {
			_ = xlsx.SetCellFormula(sheet, cell(col, row), sumFormula(v))
		}
		if style := cellStyle(bal.months[t.Format("2006-01")]); style != nil {
			style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), customNumberFormat(), style))
			_ = xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
		} else {
			style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), customNumberFormat()))
			_ = xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
		}
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	_ = xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(C%d:%c%d)", row, col-1, row))
	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontItalic(), customNumberFormat()))
	_ = xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
}

func defaultStyle() *excelize.Style {
	return &excelize.Style{
		// solid white
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFFFF"},
			Pattern: 1,
		},
	}
}

func customNumberFormat() *excelize.Style {
	fmt := "#,##0,.0"
	return &excelize.Style{
		CustomNumFmt: &fmt,
	}
}

func fontItalic() *excelize.Style {
	return &excelize.Style{
		Font: &excelize.Font{
			Italic: true,
		},
	}
}

func fontBold() *excelize.Style {
	return &excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	}
}

func fontBoldItalic() *excelize.Style {
	return &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Italic: true,
		},
	}
}

func textAlignment(a string) *excelize.Style {
	return &excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: a,
		},
	}
}

func verticalCenter() *excelize.Style {
	return &excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	}
}

func thinBorder(where ...string) *excelize.Style {
	s := &excelize.Style{}
	for _, w := range where {
		s.Border = append(s.Border, excelize.Border{
			Type:  w,
			Color: "#000000",
			Style: 1,
		})
	}
	return s
}

func thickBorder(where ...string) *excelize.Style {
	s := &excelize.Style{}
	for _, w := range where {
		s.Border = append(s.Border, excelize.Border{
			Type:  w,
			Color: "#000000",
			Style: 2,
		})
	}
	return s
}

func highlight() *excelize.Style {
	return &excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFF50"},
			Pattern: 1,
		},
	}
}

func mergeStyles(ext ...*excelize.Style) *excelize.Style {
	if len(ext) == 0 {
		return nil
	}
	for _, e := range ext[1:] {
		_ = mergo.Merge(ext[0], e, mergo.WithOverride)
	}
	return ext[0]
}

func xlsxHeaderMonths(xlsx *excelize.File, sheet string, row int, hdr string, starts, ends time.Time) {
	_ = xlsx.SetCellValue(sheet, cell('B', row), hdr)
	t := starts
	col := 'C'
	for !t.After(ends) {
		_ = xlsx.SetCellValue(sheet, cell(col, row), t.Format("2006-01"))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	_ = xlsx.SetCellValue(sheet, cell(col, row), "Total")

	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), textAlignment("right")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(col, row), style)
}

func xlsxHeader(xlsx *excelize.File, sheet string, row, cols int, hdr string) {
	_ = xlsx.SetCellValue(sheet, cell('B', row), hdr)
	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), thinBorder("bottom")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell('B'+rune(cols)+2, row), style)
}

func xlsxSumMonths(xlsx *excelize.File, sheet string, row int, hdr string, starts, ends time.Time, startRow int) {
	_ = xlsx.SetCellValue(sheet, cell('B', row), hdr)
	t := starts
	col := 'C'
	for !t.After(ends) {
		_ = xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(%c%d:%c%d)", col, startRow, col, row-1))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++

	_ = xlsx.SetCellFormula(sheet, cell(col, row), fmt.Sprintf("SUM(%c%d:%c%d)", col, startRow, col, row-1))

	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(col-1, row), style)

	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("top")))
	_ = xlsx.SetCellStyle(sheet, cell(col, row), cell(col, row), style)
}

func sumcells(col rune, rows []int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%c%d", col, rows[0])
	for _, row := range rows[1:] {
		fmt.Fprintf(&b, "+%c%d", col, row)
	}
	return b.String()
}

func xlsxSumSumMonths(xlsx *excelize.File, sheet string, row int, starts, ends time.Time, sumRows []int, accountBalances map[int]*balance, inCapital sie.Decimal) {
	_ = xlsx.SetCellValue(sheet, cell('B', row), "Resultat")

	// sum

	t := starts
	col := 'C'
	for !t.After(ends) {
		_ = xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++
	_ = xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))
	ecol := col

	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("top")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(ecol-1, row), style)
	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("top")))
	_ = xlsx.SetCellStyle(sheet, cell(ecol, row), cell(ecol, row), style)
	resultRow := row

	// eget kapital

	row++
	_ = xlsx.SetCellValue(sheet, cell('B', row), "Eget kapital")
	scol := 'C'
	for t = starts; !t.After(ends); t = t.AddDate(0, 1, 0) {
		capital := inCapital
		for _, acc := range []int{2081, 2091} {
			if month := accountBalances[acc].months[t.Format("2006-01")]; month != nil {
				for _, cv := range month {
					capital -= cv.amount
				}
			}
		}
		formula := fmt.Sprintf("%c%d", scol, resultRow)
		if scol != 'C' {
			formula += fmt.Sprintf("+%c%d", scol-1, row)
		}
		if capital != 0 {
			formula += fmt.Sprintf("+%v", capital)
		}
		_ = xlsx.SetCellFormula(sheet, cell(scol, row), formula)
		scol++
		inCapital = 0 // only add initial capital on the first iterator
	}

	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontItalic(), customNumberFormat()))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(ecol, row), style)

	// quarterly sums

	row++
	_ = xlsx.SetCellValue(sheet, cell('B', row), "Kvartalsvis resultat")
	scol = 'E'
	for t = starts.AddDate(0, 3, 0); !t.After(ends.AddDate(0, 1, 0)); t = t.AddDate(0, 3, 0) {
		_ = xlsx.SetCellFormula(sheet, cell(scol, row), fmt.Sprintf("SUM(%c%d:%c%d)", scol-2, resultRow, scol, resultRow))
		scol += 3
	}

	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat()))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(ecol, row), style)
	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBoldItalic(), customNumberFormat()))
	_ = xlsx.SetCellStyle(sheet, cell(ecol, row), cell(ecol, row), style)

	// half year sums

	row++
	_ = xlsx.SetCellValue(sheet, cell('B', row), "Halvårsvis resultat")
	scol = 'H'
	for t = starts.AddDate(0, 6, 0); !t.After(ends.AddDate(0, 1, 0)); t = t.AddDate(0, 6, 0) {
		_ = xlsx.SetCellFormula(sheet, cell(scol, row), fmt.Sprintf("SUM(%c%d:%c%d)", scol-5, resultRow, scol, resultRow))
		scol += 6
	}

	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBold(), customNumberFormat(), thickBorder("bottom")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(ecol, row), style)
	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), fontBoldItalic(), customNumberFormat(), thickBorder("bottom")))
	_ = xlsx.SetCellStyle(sheet, cell(ecol, row), cell(ecol, row), style)
}

func xlsxSectionSum(xlsx *excelize.File, sheet string, row int, hdr string, starts, ends time.Time, sumRows []int) {
	_ = xlsx.SetCellValue(sheet, cell('B', row), hdr)

	// sum

	t := starts
	col := 'C'
	for !t.After(ends) {
		_ = xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))
		col++
		t = t.AddDate(0, 1, 0)
	}
	col++
	_ = xlsx.SetCellFormula(sheet, cell(col, row), sumcells(col, sumRows))
	ecol := col

	style, _ := xlsx.NewStyle(mergeStyles(defaultStyle(), verticalCenter(), fontBold(), customNumberFormat(), thickBorder("top", "bottom")))
	_ = xlsx.SetCellStyle(sheet, cell('B', row), cell(ecol-1, row), style)
	style, _ = xlsx.NewStyle(mergeStyles(defaultStyle(), verticalCenter(), fontBoldItalic(), customNumberFormat(), thickBorder("top", "bottom")))
	_ = xlsx.SetCellStyle(sheet, cell(ecol, row), cell(ecol, row), style)
	_ = xlsx.SetRowHeight(sheet, row, 20)
}

func sumFormula(v []cellValue) string {
	var b strings.Builder
	for i, d := range v {
		switch {
		case i > 0 && d.amount >= 0:
			b.WriteString(" + ")
		case i > 0 && d.amount < 0:
			b.WriteString(" - ")
			d.amount = -d.amount
		}
		fmt.Fprintf(&b, "%v", d.amount.Float64())
	}
	return b.String()
}

func cellStyle(v []cellValue) *excelize.Style {
	if len(v) == 0 {
		return nil
	}
	var latest time.Time
	for _, d := range v {
		if d.when.After(latest) {
			latest = d.when
		}
	}
	if latest.After(time.Now().AddDate(0, 0, -7)) {
		return highlight()
	}
	return nil
}
