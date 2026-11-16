package gocodegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupIncludes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		includes func(*Scope) []Include
		reserved []string
		want     []string
	}{
		{
			name: "no collision",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("base.media", "media", false),
					gs.NewInclude("base.post", "post", false),
				}
			},
			want: []string{"media", "post"},
		},
		{
			name: "homonymous includes",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("influencer.types.post.v2.post", "post", false),
					gs.NewInclude("base.media", "media", false),
					gs.NewInclude("base.post", "post", false),
				}
			},
			want: []string{"post", "media", "post0"},
		},
		{
			name: "suffixed name never takes a base name",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("base.post", "post", false),
					gs.NewInclude("influencer.post", "post", false),
					gs.NewInclude("foo.post0", "post0", false),
				}
			},
			want: []string{"post", "post1", "post0"},
		},
		{
			name: "reserved name",
			includes: func(gs *Scope) []Include {
				return []Include{gs.NewInclude("foo.mock", "mock", false)}
			},
			reserved: []string{"mock"},
			want:     []string{"mock0"},
		},
		{
			name: "thrift package is always reserved",
			includes: func(gs *Scope) []Include {
				return []Include{gs.NewInclude("foo.thrift", "thrift", false)}
			},
			want: []string{"thrift0"},
		},
		{
			name: "predeclared identifiers are always reserved",
			includes: func(gs *Scope) []Include {
				return []Include{gs.NewInclude("foo.error", "error", false)}
			},
			want: []string{"error0"},
		},
		{
			name: "same import path is merged",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("base.post", "post", false),
					gs.NewIncludeWithPath("other.post", "post", "base/post", false),
				}
			},
			want: []string{"post"},
		},
		{
			name: "stdlib and local package with the same path",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("types.known.timestamp", "timestamp", true),
					gs.NewInclude("types.known.timestamp", "timestamp", false),
				}
			},
			want: []string{"timestamp", "timestamp0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := Scope{Reserved: tc.reserved}

			gs.Includes = tc.includes(&gs)

			assert.Equal(t, tc.want, pkgNames(DedupIncludes(&gs)))
		})
	}
}

func TestDedupIncludesLeavesScopeUntouched(t *testing.T) {
	var gs Scope

	gs.Includes = []Include{
		gs.NewInclude("base.post", "post", false),
		gs.NewInclude("influencer.post", "post", false),
		gs.NewIncludeWithPath("other.post", "post", "base/post", false),
	}

	DedupIncludes(&gs)

	assert.Equal(t, []string{"post", "post", "post"}, pkgNames(gs.Includes))
}

func TestDedupIncludesSeesIncludesAddedLater(t *testing.T) {
	gs := Scope{Reserved: []string{"mock"}}

	gs.Includes = []Include{gs.NewInclude("base.post", "post", false)}
	assert.Equal(t, []string{"post"}, pkgNames(DedupIncludes(&gs)))

	gs.Includes = append(
		gs.Includes,
		gs.NewInclude("influencer.post", "post", false),
		gs.NewInclude("foo.mock", "mock", false),
	)

	got := DedupIncludes(&gs)

	assert.Equal(t, []string{"post", "post0", "mock0"}, pkgNames(got))
	assert.Equal(t, "post", got[1].BaseName())
}

func TestImportSpec(t *testing.T) {
	gs := Scope{ImportPkgPrefix: "github.com/upfluence/"}

	gs.Includes = []Include{
		gs.NewInclude("influencer.types.post.v2.post", "post", false),
		gs.NewInclude("base.post", "post", false),
		gs.NewInclude("types.known.timestamp", "timestamp", true),
	}

	var got []string

	for _, inc := range DedupIncludes(&gs) {
		got = append(got, inc.ImportSpec())
	}

	assert.Equal(
		t,
		[]string{
			`"github.com/upfluence/influencer/types/post/v2/post"`,
			`post0 "github.com/upfluence/base/post"`,
			`"github.com/upfluence/thrift/lib/go/thrift/types/known/timestamp"`,
		},
		got,
	)
}

const (
	identityModule   = "github.com/upfluence/identity"
	identityV2Module = "github.com/upfluence/identity/v2"
)

func TestScopeImportPath(t *testing.T) {
	const scopePkgPath = "identity/types/scope/v1/scope"

	identityRewrites := map[string]string{identityModule: identityV2Module}

	for _, tt := range []struct {
		name         string
		haveRewrites map[string]string
		havePkgPath  string
		haveStdlib   bool
		want         string
	}{
		{
			name:        "no rewrites",
			havePkgPath: scopePkgPath,
			want:        "github.com/upfluence/identity/types/scope/v1/scope",
		},
		{
			name:         "sub path",
			haveRewrites: identityRewrites,
			havePkgPath:  scopePkgPath,
			want:         "github.com/upfluence/identity/v2/types/scope/v1/scope",
		},
		{
			name:         "exact match",
			haveRewrites: identityRewrites,
			havePkgPath:  "identity",
			want:         identityV2Module,
		},
		{
			name:         "sibling prefix does not match",
			haveRewrites: identityRewrites,
			havePkgPath:  "identityutil/types/util",
			want:         "github.com/upfluence/identityutil/types/util",
		},
		{
			name: "longest match wins",
			haveRewrites: map[string]string{
				identityModule: identityV2Module,
				"github.com/upfluence/identity/types/scope": "github.com/upfluence/scope/v3",
				"github.com/upfluence":                      "example.com",
			},
			havePkgPath: scopePkgPath,
			want:        "github.com/upfluence/scope/v3/v1/scope",
		},
		{
			name:         "stdlib without matching rule",
			haveRewrites: identityRewrites,
			havePkgPath:  "types/known/timestamp",
			haveStdlib:   true,
			want:         "github.com/upfluence/thrift/lib/go/thrift/types/known/timestamp",
		},
		{
			name: "stdlib with matching rule",
			haveRewrites: map[string]string{
				"github.com/upfluence/thrift": "github.com/upfluence/thrift/v2",
			},
			havePkgPath: "types/known/timestamp",
			haveStdlib:  true,
			want:        "github.com/upfluence/thrift/v2/lib/go/thrift/types/known/timestamp",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gs := Scope{
				ImportPkgPrefix: "github.com/upfluence/",
				ImportRewrites:  tt.haveRewrites,
			}

			assert.Equal(t, tt.want, gs.ImportPath(tt.havePkgPath, tt.haveStdlib))
		})
	}
}

func TestIncludePkgPathAppliesImportRewrites(t *testing.T) {
	gs := Scope{
		ImportPkgPrefix: "github.com/upfluence/",
		ImportRewrites:  map[string]string{identityModule: identityV2Module},
	}

	gs.Includes = []Include{
		gs.NewInclude("identity.types.scope.v1.scope", "scope", false),
		gs.NewInclude("identityutil.types.util", "util", false),
	}

	var got []string

	for _, inc := range DedupIncludes(&gs) {
		got = append(got, inc.ImportSpec())
	}

	assert.Equal(
		t,
		[]string{
			`"github.com/upfluence/identity/v2/types/scope/v1/scope"`,
			`"github.com/upfluence/identityutil/types/util"`,
		},
		got,
	)
}

func pkgNames(incs []Include) []string {
	res := make([]string, 0, len(incs))

	for _, inc := range incs {
		res = append(res, inc.PkgName)
	}

	return res
}
