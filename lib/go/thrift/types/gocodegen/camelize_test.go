package gocodegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicize(t *testing.T) {
	for _, tc := range []struct {
		name string
		have string
		want string
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
		{
			name: "new prefix escaped",
			have: "new_service",
			want: "NewService_",
		},
		{
			name: "args suffix escaped",
			have: "request_args",
			want: "RequestArgs_",
		},
		{
			name: "result suffix escaped",
			have: "request_result",
			want: "RequestResult_",
		},
		{
			name: "sink suffix escaped",
			have: "async_sink",
			want: "AsyncSink_",
		},
		{
			name: "stream suffix escaped",
			have: "event_stream",
			want: "EventStream_",
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

func TestPublicizeHelper(t *testing.T) {
	for _, tt := range []struct {
		name string
		have string
		want string
	}{
		{
			name: "args suffix is not escaped",
			have: "fetch_stream_args",
			want: "FetchStreamArgs",
		},
		{
			name: "result suffix is not escaped",
			have: "fetch_stream_result",
			want: "FetchStreamResult",
		},
		{
			name: "new prefix remains escaped",
			have: "new_stream_args",
			want: "NewStreamArgs_",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PublicizeHelper(tt.have))
		})
	}
}

func TestVariableName(t *testing.T) {
	for _, tt := range []struct {
		name string
		have string
		want string
	}{
		{
			name: "regular name",
			have: "body",
			want: "body",
		},
		{
			name: "keyword",
			have: "type",
			want: "type_a1",
		},
		{
			name: "uppercase keyword",
			have: "Type",
			want: "type_a1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VariableName(tt.have))
		})
	}
}

func TestPublicizeField(t *testing.T) {
	for _, tt := range []struct {
		name string
		have string
		want string
	}{
		{
			name: "regular field",
			have: "user_id",
			want: "UserID",
		},
		{
			name: "trailing underscore",
			have: "fields_",
			want: "Fields_",
		},
		{
			name: "reserved suffix",
			have: "request_args",
			want: "RequestArgs_",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PublicizeField(tt.have))
		})
	}
}
