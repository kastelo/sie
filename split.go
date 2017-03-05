package sie

import (
	"bufio"
	"strconv"
	"strings"
	"unicode/utf8"
)

func splitWords(s string) []string {
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Split(scanWords)
	var res []string
	for sc.Scan() {
		word, _ := strconv.Unquote(`"` + sc.Text() + `"`)
		res = append(res, word)
	}
	return res
}

func scanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}

	// Check for leading quote
	inQuote := false
	if r, width := utf8.DecodeRune(data[start:]); r == '"' {
		start += width
		inQuote = true
	}

	// Scan until space or end quote, marking end of word.
	inEscape := false
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if !inEscape && r == '\\' {
			inEscape = true
			continue
		}
		if !inQuote && isSpace(r) || !inEscape && inQuote && r == '"' {
			return i + width, data[start:i], nil
		}
		inEscape = false
	}

	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}

	// Request more data.
	return start, nil, nil
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t':
		return true
	default:
		return false
	}
}
