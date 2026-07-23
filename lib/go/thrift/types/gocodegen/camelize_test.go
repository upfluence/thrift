package gocodegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicize(t *testing.T) {
	for _, tc := range []struct {
		name  string
		have  string
		want  string
	}{
		{
			name: "simple snake_case",
			have: "foo_bar",
			want: "FooBar",
		},
		{
			name: "initialism at start",
			have: "http_url",
			want: "HTTPURL",
		},
		{
			name: "initialism in middle",
			have: "get_http_url",
			want: "GetHTTPURL",
		},
		{
			name: "no underscore",
			have: "foo",
			want: "Foo",
		},
		{
			name: "single initialism",
			have: "id",
			want: "ID",
		},
		{
			name: "trailing underscore ignored",
			have: "foo_",
			want: "Foo",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Publicize(tc.have))
		})
	}
}

func TestPrivatize(t *testing.T) {
	for _, tc := range []struct {
		name string
		have string
		want string
	}{
		{
			name: "simple snake_case",
			have: "foo_bar",
			want: "fooBar",
		},
		{
			name: "initialism at start kept lowercase",
			have: "http_url",
			want: "httpURL",
		},
		{
			name: "initialism in middle uppercased",
			have: "get_http_url",
			want: "getHTTPURL",
		},
		{
			name: "no underscore",
			have: "foo",
			want: "foo",
		},
		{
			name: "single initialism at start",
			have: "id",
			want: "id",
		},
		{
			name: "initialism only in second word",
			have: "some_id",
			want: "someID",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Privatize(tc.have))
		})
	}
}
