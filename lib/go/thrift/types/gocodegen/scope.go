package gocodegen

import (
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultThriftImport = "github.com/upfluence/thrift/lib/go/thrift"

// PredeclaredIdentifiers lists the Go universe scope identifiers, which an
// import must not shadow in generated code.
var PredeclaredIdentifiers = []string{
	"any", "bool", "byte", "comparable", "complex64", "complex128", "error",
	"float32", "float64", "int", "int8", "int16", "int32", "int64", "rune",
	"string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"true", "false", "iota", "nil",
	"append", "cap", "clear", "close", "complex", "copy", "delete", "imag",
	"len", "make", "max", "min", "new", "panic", "print", "println", "real",
	"recover",
}

// Include holds the resolved Go package information for a single thrift include.
type Include struct {
	// Namespace is the wildcard namespace of the included program, e.g. "base.page".
	Namespace string

	// PkgName is the Go package identifier, e.g. "page". It differs from
	// BaseName when DedupIncludes had to alias the package.
	PkgName string

	// GoPkgPath is the Go package path without the configured import prefix.
	GoPkgPath string

	// Stdlib indicates the include is from the thrift standard library.
	Stdlib bool

	// scope is a back-pointer to the owning Scope for prefix resolution.
	scope *Scope

	baseName string
}

// BaseName returns the package name declared by the included Go package,
// before any deduplication.
func (inc Include) BaseName() string {
	if inc.baseName != "" {
		return inc.baseName
	}

	return inc.PkgName
}

// PkgPath returns the fully-qualified Go import path for this include.
func (inc Include) PkgPath() string {
	if inc.Stdlib {
		return inc.scope.ThriftImport() + "/" + inc.goPkgPath()
	}

	return inc.scope.ImportPkgPrefix + inc.goPkgPath()
}

func (inc Include) goPkgPath() string {
	if inc.GoPkgPath != "" {
		return inc.GoPkgPath
	}

	return strings.ReplaceAll(inc.Namespace, ".", "/")
}

// ImportSpec returns the import declaration of the include, aliased only when
// PkgName differs from the last element of the import path.
func (inc Include) ImportSpec() string {
	p := inc.PkgPath()

	if inc.PkgName == path.Base(p) {
		return strconv.Quote(p)
	}

	return inc.PkgName + " " + strconv.Quote(p)
}

// Scope holds the Go-specific compilation context derived from a plugin
// GenerateCodeRequest's options.
type Scope struct {
	// ThriftImportPath is the full import path of the thrift runtime package.
	ThriftImportPath string

	// ImportPkgPrefix is the module path prefix used to build fully-qualified
	// import paths for generated packages, e.g. "github.com/upfluence/".
	ImportPkgPrefix string

	// LocalPkg is the namespace of the program being compiled,
	LocalPkg string

	// Includes holds resolved package info for each direct include of the program.
	Includes []Include

	// Reserved lists the names DedupIncludes keeps away from includes, besides
	// the thrift package and the Go predeclared identifiers.
	Reserved []string
}

// ThriftPkg returns the local package identifier for the thrift runtime import.
func (gs Scope) ThriftPkg() string {
	return filepath.Base(gs.ThriftImport())
}

// NewInclude constructs an Include with a back-pointer to gs.
func (gs *Scope) NewInclude(namespace, pkgName string, stdlib bool) Include {
	return gs.NewIncludeWithPath(namespace, pkgName, "", stdlib)
}

// NewIncludeWithPath constructs an Include with an explicit Go package path.
func (gs *Scope) NewIncludeWithPath(namespace, pkgName, goPkgPath string, stdlib bool) Include {
	return Include{
		Namespace: namespace,
		PkgName:   pkgName,
		GoPkgPath: goPkgPath,
		Stdlib:    stdlib,
		scope:     gs,
		baseName:  pkgName,
	}
}

// DedupIncludes returns the includes of gs with one entry per import path,
// each with a PkgName distinct from the others, from gs.Reserved, from the
// thrift package and from the Go predeclared identifiers. gs is left untouched.
func DedupIncludes(gs *Scope) []Include {
	used := make(map[string]struct{})

	for _, names := range [][]string{{gs.ThriftPkg()}, PredeclaredIdentifiers, gs.Reserved} {
		for _, name := range names {
			used[name] = struct{}{}
		}
	}

	seen := make(map[string]struct{}, len(gs.Includes))
	res := make([]Include, 0, len(gs.Includes))

	for _, inc := range gs.Includes {
		key := inc.dedupKey()

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		inc.baseName = inc.BaseName()
		inc.PkgName = ""

		if _, ok := used[inc.baseName]; !ok {
			used[inc.baseName] = struct{}{}
			inc.PkgName = inc.baseName
		}

		res = append(res, inc)
	}

	// Base names are all claimed above, so a suffixed name never takes one.
	for i := range res {
		if res[i].PkgName == "" {
			res[i].PkgName = uniqueName(res[i].baseName, used)
			used[res[i].PkgName] = struct{}{}
		}
	}

	return res
}

func (inc Include) dedupKey() string {
	return strconv.FormatBool(inc.Stdlib) + ":" + inc.goPkgPath()
}

func uniqueName(base string, used map[string]struct{}) string {
	for i := 0; ; i++ {
		name := base + strconv.Itoa(i)

		if _, ok := used[name]; !ok {
			return name
		}
	}
}

// ThriftImport returns the full import path of the thrift runtime package for
// the given scope. When no ImportPkgPrefix is set it falls back to
// DefaultThriftImport.
func (gs Scope) ThriftImport() string {
	if gs.ThriftImportPath != "" {
		return gs.ThriftImportPath
	}

	if gs.ImportPkgPrefix == "" {
		return DefaultThriftImport
	}

	return gs.ImportPkgPrefix + "thrift/lib/go/thrift"
}
