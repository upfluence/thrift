package plugin

import (
	"strings"

	"github.com/upfluence/thrift/lib/go/thrift/types/gocodegen"
	"github.com/upfluence/thrift/lib/go/thrift/types/program_definition"
)

const importRewriteOptionPrefix = "import_rewrite:"

// BuildScope extracts the Go-specific scope from a GenerateCodeRequest.
func BuildGoScope(req *GenerateCodeRequest) gocodegen.Scope {
	thriftImport := req.Options["thrift_import"]

	if thriftImport == "" {
		thriftImport = gocodegen.DefaultThriftImport
	}

	gs := gocodegen.Scope{
		ThriftImportPath: thriftImport,
		ImportPkgPrefix:  req.Options["package_prefix"],
		ImportRewrites:   buildImportRewrites(req.Options),
		LocalPkg:         req.Program.Namespaces["*"],
	}

	includes := make([]gocodegen.Include, 0, len(req.Program.Includes))

	for _, inc := range req.Program.Includes {
		includes = append(includes, gs.NewIncludeWithPath(
			inc.Namespaces["*"],
			program_definition.GoPackageName(inc),
			program_definition.GoPackagePath(inc),
			inc.Stdlib,
		))
	}

	gs.Includes = includes

	return gs
}

func buildImportRewrites(opts map[string]string) map[string]string {
	var rewrites map[string]string

	for k, v := range opts {
		from, ok := strings.CutPrefix(k, importRewriteOptionPrefix)

		if !ok {
			continue
		}

		if rewrites == nil {
			rewrites = make(map[string]string)
		}

		rewrites[from] = v
	}

	return rewrites
}
