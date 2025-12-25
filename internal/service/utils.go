package service

import (
	"strings"
	"unicode/utf8"
)

// sanitizeUTF8 removes invalid UTF-8 sequences from string
// This prevents PostgreSQL encoding errors when saving text
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Remove invalid UTF-8 sequences
	var result strings.Builder
	result.Grow(len(s))

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 sequence, skip this byte
			s = s[1:]
			continue
		}
		result.WriteRune(r)
		s = s[size:]
	}

	return result.String()
}
