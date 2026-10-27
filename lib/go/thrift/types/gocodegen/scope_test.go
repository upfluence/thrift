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
			reserved: []string{"thrift"},
			want:     []string{"media", "post"},
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
				return []Include{gs.NewInclude("foo.thrift", "thrift", false)}
			},
			reserved: []string{"thrift"},
			want:     []string{"thrift0"},
		},
		{
			name: "same import path shares one name",
			includes: func(gs *Scope) []Include {
				return []Include{
					gs.NewInclude("base.post", "post", false),
					gs.NewIncludeWithPath("other.post", "post", "base/post", false),
				}
			},
			want: []string{"post", "post"},
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
			var gs Scope

			gs.Includes = tc.includes(&gs)
			gs.DedupIncludes(tc.reserved...)

			assert.Equal(t, tc.want, pkgNames(gs.Includes))
		})
	}
}

func TestDedupIncludesKeepsPreviousReservations(t *testing.T) {
	var gs Scope

	gs.Includes = []Include{
		gs.NewInclude("base.post", "post", false),
		gs.NewInclude("influencer.post", "post", false),
		{Namespace: "foo.error", PkgName: "error"},
	}

	gs.DedupIncludes("error")
	gs.DedupIncludes("mock")

	assert.Equal(t, []string{"post", "post0", "error0"}, pkgNames(gs.Includes))
	assert.Equal(t, "error", gs.Includes[2].BaseName())
}

func TestImportSpec(t *testing.T) {
	gs := Scope{ImportPkgPrefix: "github.com/upfluence/"}

	gs.Includes = []Include{
		gs.NewInclude("influencer.types.post.v2.post", "post", false),
		gs.NewInclude("base.post", "post", false),
		gs.NewInclude("types.known.timestamp", "timestamp", true),
		gs.NewIncludeWithPath("other.post", "post", "base/post", false),
	}
	gs.DedupIncludes()

	var got []string

	for _, inc := range gs.Imports() {
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

func pkgNames(incs []Include) []string {
	res := make([]string, 0, len(incs))

	for _, inc := range incs {
		res = append(res, inc.PkgName)
	}

	return res
}
