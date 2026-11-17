package goinitialism

import "strings"

// Initialisms are the common initialisms of Go identifiers, in their
// canonical spelling, taken from
// https://github.com/mgechev/revive/blob/master/internal/rule/name.go and
// mirrored in compiler/cpp/src/thrift/generate/t_go_generator.cc and in the
// ruby text codec.
var Initialisms = []string{
	"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP",
	"HTTPS", "ID", "IDs", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC",
	"SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID",
	"UUID", "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS",
}

var canonicals = func() map[string]string {
	m := make(map[string]string, len(Initialisms))

	for _, i := range Initialisms {
		m[strings.ToUpper(i)] = i
	}

	return m
}()

// Lookup returns the canonical spelling of word if it is an initialism,
// matched case insensitively: "ids" is "IDs".
func Lookup(word string) (string, bool) {
	c, ok := canonicals[strings.ToUpper(word)]

	return c, ok
}
