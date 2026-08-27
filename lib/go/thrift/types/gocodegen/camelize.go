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

var goKeywords = map[string]struct{}{
	"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {},
	"default": {}, "defer": {}, "else": {}, "error": {}, "fallthrough": {},
	"for": {}, "func": {}, "go": {}, "goto": {}, "if": {}, "import": {},
	"interface": {}, "map": {}, "package": {}, "range": {}, "return": {},
	"select": {}, "struct": {}, "switch": {}, "type": {}, "var": {},
}

// Publicize converts a snake_case identifier to an exported Go identifier
// (UpperCamelCase), applying common Go initialisms (e.g. "http_url" →
// "HTTPURL").
func Publicize(s string) string {
	name := camelize(s, true)

	if strings.HasPrefix(name, "New") ||
		strings.HasSuffix(name, "Args") ||
		strings.HasSuffix(name, "Result") ||
		strings.HasSuffix(name, "Sink") ||
		strings.HasSuffix(name, "Stream") {
		return name + "_"
	}

	return name
}

// PublicizeHelper converts an implicit service helper identifier, such as
// method_args, without escaping its reserved suffix.
func PublicizeHelper(s string) string {
	name := camelize(s, true)

	if strings.HasPrefix(name, "New") {
		return name + "_"
	}

	return name
}

// PublicizeField converts an IDL field name to its exported Go identifier.
func PublicizeField(s string) string {
	name := Publicize(s)

	if strings.HasSuffix(s, "_") && !strings.HasSuffix(name, "_") {
		return name + "_"
	}

	return name
}

// Privatize converts a snake_case identifier to an unexported Go identifier
// (lowerCamelCase), applying common Go initialisms on every word except the
// first (e.g. "http_url" → "httpURL").
func Privatize(s string) string {
	return camelize(s, false)
}

// VariableName escapes Go keywords using the same suffix as the Go compiler.
func VariableName(s string) string {
	if _, ok := goKeywords[strings.ToLower(s)]; ok {
		return strings.ToLower(s) + "_a1"
	}

	return s
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
