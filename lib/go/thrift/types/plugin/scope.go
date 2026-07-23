package plugin

import (
	"path/filepath"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/types/gocodegen"
)

// BuildScope extracts the Go-specific scope from a GenerateCodeRequest.
func BuildGoScope(req *GenerateCodeRequest) gocodegen.Scope {
	thriftImport := req.Options["thrift_import"]

	if thriftImport == "" {
		thriftImport = gocodegen.DefaultThriftImport
	}

	gs := gocodegen.Scope{
		ThriftPkg:       filepath.Base(thriftImport),
		ImportPkgPrefix: req.Options["package_prefix"],
		LocalPkg:        req.Program.Namespaces["*"],
	}

	includes := make([]gocodegen.Include, 0, len(req.Program.Includes))

	for _, inc := range req.Program.Includes {
		includes = append(includes, gs.NewInclude(
			inc.Namespaces["*"],
			filepath.Base(strings.ReplaceAll(inc.Namespaces["*"], ".", "/")),
			inc.Stdlib,
		))
	}

	gs.Includes = includes

	return gs
}
