package gocodegen

import (
	"path/filepath"
	"strings"
)

const DefaultThriftImport = "github.com/upfluence/thrift/lib/go/thrift"

// Include holds the resolved Go package information for a single thrift include.
type Include struct {
	// Namespace is the wildcard namespace of the included program, e.g. "base.page".
	Namespace string

	// PkgName is the Go package identifier, e.g. "page".
	PkgName string

	// GoPkgPath is the Go package path without the configured import prefix.
	GoPkgPath string

	// Stdlib indicates the include is from the thrift standard library.
	Stdlib bool

	// scope is a back-pointer to the owning Scope for prefix resolution.
	scope *Scope
}

// PkgPath returns the fully-qualified Go import path for this include.
func (inc Include) PkgPath() string {
	path := inc.GoPkgPath

	if path == "" {
		path = strings.ReplaceAll(inc.Namespace, ".", "/")
	}

	if inc.Stdlib {
		return inc.scope.ThriftImport() + "/" + path
	}

	return inc.scope.ImportPkgPrefix + path
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
