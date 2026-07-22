package pluginutil

import (
	"path/filepath"
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/types/goscope"
	"github.com/upfluence/thrift/lib/go/thrift/types/plugin"
)

// GoScope is re-exported from goscope for convenience.
type GoScope = goscope.GoScope

const defaultThriftImport = "github.com/upfluence/thrift/lib/go/thrift"

// ThriftImport returns the full import path of the thrift runtime package.
func ThriftImport(gs GoScope) string {
	if gs.ImportPkgPrefix == "" {
		return defaultThriftImport
	}

	return gs.ImportPkgPrefix + "thrift/lib/go/thrift"
}

// BuildGoScope extracts the Go-specific scope from a GenerateCodeRequest.
func BuildGoScope(req *plugin.GenerateCodeRequest) GoScope {
	thriftImport := req.Options["thrift_import"]

	if thriftImport == "" {
		thriftImport = defaultThriftImport
	}

	gs := GoScope{
		ThriftPkg:       filepath.Base(thriftImport),
		ImportPkgPrefix: req.Options["package_prefix"],
		LocalPkg:        req.Program.Namespaces["*"],
	}

	includes := make([]goscope.Include, 0, len(req.Program.Includes))

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
