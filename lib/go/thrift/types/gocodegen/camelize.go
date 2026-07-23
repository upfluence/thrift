package gocodegen

import (
	"strings"
	"unicode"
)

// commonInitialisms is the set of well-known Go initialisms taken from
// https://github.com/golang/lint/blob/master/lint.go#L692 and mirrored in
// compiler/cpp/src/thrift/generate/t_go_generator.cc.
var commonInitialisms = map[string]struct{}{
	"API":   {},
	"ASCII": {},
	"CPU":   {},
	"CSS":   {},
	"DNS":   {},
	"EOF":   {},
	"GUID":  {},
	"HTML":  {},
	"HTTP":  {},
	"HTTPS": {},
	"ID":    {},
	"IP":    {},
	"JSON":  {},
	"LHS":   {},
	"QPS":   {},
	"RAM":   {},
	"RHS":   {},
	"RPC":   {},
	"SLA":   {},
	"SMTP":  {},
	"SSH":   {},
	"TCP":   {},
	"TLS":   {},
	"TTL":   {},
	"UDP":   {},
	"UI":    {},
	"UID":   {},
	"UUID":  {},
	"URI":   {},
	"URL":   {},
	"UTF8":  {},
	"VM":    {},
	"XML":   {},
	"XSRF":  {},
	"XSS":   {},
}

// Publicize converts a snake_case identifier to an exported Go identifier
// (UpperCamelCase), applying common Go initialisms (e.g. "http_url" →
// "HTTPURL").
func Publicize(s string) string {
	return camelize(s, true)
}

// Privatize converts a snake_case identifier to an unexported Go identifier
// (lowerCamelCase), applying common Go initialisms on every word except the
// first (e.g. "http_url" → "httpURL").
func Privatize(s string) string {
	return camelize(s, false)
}

// camelize implements the shared camelization logic used by Publicize and
// Privatize. When public is true the first word is also capitalised /
// initialism-expanded; when false it is left lowercase.
func camelize(s string, public bool) string {
	parts := strings.Split(s, "_")
	var b strings.Builder

	for i, part := range parts {
		if part == "" {
			continue
		}

		upper := strings.ToUpper(part)

		if _, ok := commonInitialisms[upper]; ok {
			if i == 0 && !public {
				// First word, private: keep fully lowercase.
				b.WriteString(strings.ToLower(part))
			} else {
				b.WriteString(upper)
			}

			continue
		}

		runes := []rune(part)

		if i == 0 && !public {
			// First word, private: keep first letter lowercase.
			b.WriteString(string(runes))
		} else {
			b.WriteRune(unicode.ToUpper(runes[0]))
			b.WriteString(string(runes[1:]))
		}
	}

	return b.String()
}
