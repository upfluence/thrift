package wordcase

import (
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/internal/goinitialism"
)

// Case is a naming convention.
type Case int

const (
	// SnakeCase renders PaypalAPI as "paypal_api".
	SnakeCase Case = iota
	// KebabCase renders PaypalAPI as "paypal-api".
	KebabCase
	// ScreamingSnakeCase renders PaypalAPI as "PAYPAL_API".
	ScreamingSnakeCase
	// CamelCase renders PaypalAPI as "paypalApi", or "paypalAPI" with the
	// API initialism.
	CamelCase
	// PascalCase renders paypal_api as "PaypalApi", or "PaypalAPI" with the
	// API initialism.
	PascalCase
	// LowerCase renders PaypalAPI as "paypalapi".
	LowerCase
	// IdentityCase renders the string as is.
	IdentityCase
)

// Caser renders ASCII identifiers in a naming convention: the identifier is
// split in words, see SplitWords, each word is cased then the words are
// joined. CamelCase and PascalCase render the Go initialisms as such, e.g.
// "API" or "IDs". A Caser is safe for concurrent use.
type Caser struct {
	c Case
}

func NewCaser(c Case) Caser {
	return Caser{c: c}
}

func (c Caser) String(s string) string {
	if c.c == IdentityCase {
		return s
	}

	words := SplitWords(s)

	for i, w := range words {
		switch c.c {
		case ScreamingSnakeCase:
			words[i] = strings.ToUpper(w)
		case CamelCase, PascalCase:
			words[i] = c.camelWord(w, i == 0 && c.c == CamelCase)
		default:
			words[i] = strings.ToLower(w)
		}
	}

	switch c.c {
	case KebabCase:
		return strings.Join(words, "-")
	case CamelCase, PascalCase, LowerCase:
		return strings.Join(words, "")
	}

	return strings.Join(words, "_")
}

func (c Caser) camelWord(w string, first bool) string {
	if first {
		return strings.ToLower(w)
	}

	if i, ok := goinitialism.Lookup(w); ok {
		return i
	}

	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

// SplitWords splits the ASCII identifier s in words on underscores, on lower
// to upper case changes and at the end of acronyms: "PaypalAPI" is "Paypal"
// and "API", "HTTPServer" is "HTTP" and "Server". Digits stick to the
// preceding word, and the Go initialisms ending in lower case are kept whole:
// "UserIDs" is "User" and "IDs".
func SplitWords(s string) []string {
	var (
		words []string
		start int
	)

	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '_':
			words = appendWord(words, s[start:i])
			start = i + 1
		case i > start && isUpper(s[i]) && isWordStart(s, start, i):
			words = appendWord(words, s[start:i])
			start = i
		}
	}

	return appendWord(words, s[start:])
}

// isWordStart reports whether the upper case byte at i starts a word: it
// follows a lower case letter or a digit, or it ends an acronym which is not
// the start of an initialism such as "IDs".
func isWordStart(s string, start, i int) bool {
	prev := s[i-1]

	if isLower(prev) || isDigit(prev) {
		return true
	}

	if !isUpper(prev) || i+1 >= len(s) || !isLower(s[i+1]) {
		return false
	}

	end := i + 1

	for end < len(s) && isLower(s[end]) {
		end++
	}

	initialism, ok := goinitialism.Lookup(s[start:end])

	return !ok || initialism != s[start:end]
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
func isLower(b byte) bool { return b >= 'a' && b <= 'z' }
func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func appendWord(words []string, word string) []string {
	if word == "" {
		return words
	}

	return append(words, word)
}
