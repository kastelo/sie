package main

import (
	"log/slog"
	"os"

	"kastelo.dev/sie"
	"kastelo.dev/sie/excel"
)

func main() {
	doc, err := sie.Parse(os.Stdin)
	if err != nil {
		slog.Error("Error parsing SIE file", "error", err)
	}

	bs, err := excel.ResultXLSX(doc)
	if err != nil {
		slog.Error("Error creating Excel file", "error", err)
	}
	if err := os.WriteFile("result.xlsx", bs, 0o644); err != nil {
		slog.Error("Error writing Excel file", "error", err)
	}
}
