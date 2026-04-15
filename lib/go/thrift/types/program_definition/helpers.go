package program_definition

import (
	"path/filepath"
	"strings"
)

func GoPackageName(p *ProgramDefinition) string {
	return goParseValue(
		p,
		func(key string) string {
			ks := strings.Split(key, ".")

			return ks[len(ks)-1]
		},
	)
}

func GoPackagePath(p *ProgramDefinition) string {
	return goParseValue(
		p,
		func(key string) string {
			return strings.ReplaceAll(key, ".", string(filepath.Separator))
		},
	)
}

func goParseValue(p *ProgramDefinition, fn func(key string) string) string {
	for _, key := range []string{"go", "*"} {
		v := p.Namespaces[key]

		if v == "" {
			continue
		}

		if res := fn(v); res != "" {
			return res
		}
	}

	return strings.TrimSuffix(filepath.Base(p.Path), ".thrift")
}
