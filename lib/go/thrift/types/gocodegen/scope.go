package gocodegen

import "strings"

const DefaultThriftImport = "github.com/upfluence/thrift/lib/go/thrift"

// Include holds the resolved Go package information for a single thrift include.
type Include struct {
	// Namespace is the wildcard namespace of the included program, e.g. "base.page".
	Namespace string

	// PkgName is the Go package identifier, e.g. "page".
	PkgName string

	// Stdlib indicates the include is from the thrift standard library.
	Stdlib bool

	// scope is a back-pointer to the owning Scope for prefix resolution.
	scope *Scope
}

// PkgPath returns the fully-qualified Go import path for this include.
func (inc Include) PkgPath() string {
	path := strings.ReplaceAll(inc.Namespace, ".", "/")

	if inc.Stdlib {
		return inc.scope.ThriftPkg + "/" + path
	}

	return inc.scope.ImportPkgPrefix + path
}

// Scope holds the Go-specific compilation context derived from a plugin
// GenerateCodeRequest's options.
type Scope struct {
	// ThriftPkg is the local package identifier for the thrift runtime import,
	// e.g. "thrift" for "github.com/upfluence/thrift/lib/go/thrift".
	ThriftPkg string

	// ImportPkgPrefix is the module path prefix used to build fully-qualified
	// import paths for generated packages, e.g. "github.com/upfluence/".
	ImportPkgPrefix string

	// LocalPkg is the namespace of the program being compiled,
	LocalPkg string

	// Includes holds resolved package info for each direct include of the program.
	Includes []Include
}

// NewInclude constructs an Include with a back-pointer to gs.
func (gs *Scope) NewInclude(namespace, pkgName string, stdlib bool) Include {
	return Include{
		Namespace: namespace,
		PkgName:   pkgName,
		Stdlib:    stdlib,
		scope:     gs,
	}
}

// ThriftImport returns the full import path of the thrift runtime package for
// the given scope. When no ImportPkgPrefix is set it falls back to
// DefaultThriftImport.
func (gs Scope) ThriftImport() string {
	if gs.ImportPkgPrefix == "" {
		return DefaultThriftImport
	}

	return gs.ImportPkgPrefix + "thrift/lib/go/thrift"
}
