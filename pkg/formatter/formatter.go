package formatter

import (
	"strconv"
	"strings"
)

// FormatNumber converts an integer to a string with commas as thousands separators.
// Example: 1234567 -> "1,234,567"
func FormatNumber(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		s = s[1:]
	}

	le := len(s)
	if le <= 3 {
		if n < 0 {
			return "-" + s
		}
		return s
	}

	sepCount := (le - 1) / 3

	res := make([]byte, le+sepCount)

	j := len(res) - 1
	for i := le - 1; i >= 0; i-- {
		res[j] = s[i]
		j--
		if (le-i)%3 == 0 && i > 0 {
			res[j] = ','
			j--
		}
	}

	if n < 0 {
		return "-" + string(res)
	}
	return string(res)
}

// EscapeMarkdownV2 escapes special characters in Markdown V2 format
func EscapeMarkdownV2(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			sb.WriteRune('\\')
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
