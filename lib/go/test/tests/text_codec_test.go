package tests

import (
	"database/sql"
	"encoding/json"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/types/annotation/text"

	"github.com/upfluence/thrift/lib/go/test/gen/texttest"
)

func TestTextMarshalStruct(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    thrift.RegistrableStruct
		want    string
		wantErr error
	}{
		{
			name: "enum and integer",
			have: &texttest.Ownership{Type: texttest.Type_PaypalAPI, EntityID: 42},
			want: "paypal_api:42",
		},
		{
			name: "escaped braces",
			have: &texttest.Handle{Media: texttest.MediaType_MediaYoutubeShort, Name: "bob", Shard: -3},
			want: "youtube/bob@{-3}",
		},
		{
			name:    "ambiguous string",
			have:    &texttest.Handle{Name: "a@{b"},
			wantErr: text.ErrInvalidValue,
		},
		{
			name:    "unknown enum value",
			have:    &texttest.Ownership{Type: texttest.Type(42)},
			wantErr: text.ErrInvalidValue,
		},
		{
			name:    "nil pointer",
			have:    (*texttest.Ownership)(nil),
			wantErr: text.ErrInvalidValue,
		},
		{
			name:    "no Format annotation",
			have:    &texttest.Unannotated{ID: 1},
			wantErr: text.ErrInvalidDefinition,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := text.MarshalStruct(tt.have)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTextUnmarshalStruct(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    string
		want    *texttest.Ownership
		wantErr error
	}{
		{
			name: "text form",
			have: "gmail:7",
			want: &texttest.Ownership{Type: texttest.Type_GoogleOAuth, EntityID: 7},
		},
		{
			name: "thrift name",
			have: "Type_Team:-1",
			want: &texttest.Ownership{Type: texttest.Type_Team, EntityID: -1},
		},
		{name: "unknown enum text", have: "nope:1", wantErr: text.ErrInvalidFormat},
		{name: "invalid integer", have: "team:1.5", wantErr: text.ErrInvalidFormat},
		{name: "missing separator", have: "team", wantErr: text.ErrInvalidFormat},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := text.UnmarshalStruct[texttest.Ownership](tt.have)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTextEnum(t *testing.T) {
	s, err := text.MarshalEnum(texttest.MediaType_MediaInstagram)
	require.NoError(t, err)
	assert.Equal(t, "instagram", s)

	v, err := text.UnmarshalEnum[texttest.MediaType]("tt")
	require.NoError(t, err)
	assert.Equal(t, texttest.MediaType_MediaTiktok, v)

	_, err = text.UnmarshalEnum[texttest.MediaType]("MediaTiktok")
	require.ErrorIs(t, err, text.ErrInvalidFormat)

	s, err = text.MarshalEnum(texttest.Plain_FooBar)
	require.NoError(t, err)
	assert.Equal(t, "foo_bar", s)
}

type textDocument struct {
	Owner text.Struct[texttest.Ownership] `json:"owner"`
	Media text.Enum[texttest.MediaType]   `json:"media"`
}

func TestTextWrappersJSON(t *testing.T) {
	doc := textDocument{
		Owner: text.Struct[texttest.Ownership]{
			V: texttest.Ownership{Type: texttest.Type_Company, EntityID: 1},
		},
		Media: text.Enum[texttest.MediaType]{V: texttest.MediaType_MediaTiktok},
	}

	b, err := json.Marshal(doc)
	require.NoError(t, err)
	assert.JSONEq(t, `{"owner":"company:1","media":"tt"}`, string(b))

	var got textDocument

	require.NoError(t, json.Unmarshal([]byte(`{"owner":"Company:1","media":"tt"}`), &got))
	assert.Equal(t, doc, got)

	assert.ErrorIs(
		t,
		json.Unmarshal([]byte(`{"owner":"company"}`), &got),
		text.ErrInvalidFormat,
	)
}

func TestTextWrappersSQL(t *testing.T) {
	var owner sql.Null[text.Struct[texttest.Ownership]]

	require.NoError(t, owner.Scan([]byte("team:3")))
	assert.Equal(
		t,
		sql.Null[text.Struct[texttest.Ownership]]{
			V:     text.Struct[texttest.Ownership]{V: texttest.Ownership{Type: texttest.Type_Team, EntityID: 3}},
			Valid: true,
		},
		owner,
	)

	v, err := owner.Value()
	require.NoError(t, err)
	assert.Equal(t, "team:3", v)

	require.NoError(t, owner.Scan(nil))
	assert.False(t, owner.Valid)

	var media text.Enum[texttest.MediaType]

	assert.ErrorIs(t, media.Scan(42), text.ErrInvalidFormat)
}

func TestTextWrappersFlag(t *testing.T) {
	var (
		owner text.Struct[texttest.Ownership]
		media text.Enum[texttest.MediaType]

		fs = flag.NewFlagSet("test", flag.ContinueOnError)
	)

	fs.Var(&owner, "owner", "")
	fs.Var(&media, "media", "")

	require.NoError(t, fs.Parse([]string{"-owner", "gmail:9", "-media", "youtube"}))
	assert.Equal(t, "gmail:9", owner.String())
	assert.Equal(t, "youtube", media.String())
}

func TestTextStructInvalidType(t *testing.T) {
	var s text.Struct[texttest.Unannotated]

	_, err := s.MarshalText()
	require.ErrorIs(t, err, text.ErrInvalidDefinition)
	require.ErrorIs(t, s.Set("1"), text.ErrInvalidDefinition)

	var i text.Struct[int]

	_, err = i.MarshalText()
	require.ErrorIs(t, err, text.ErrInvalidDefinition)
}

func TestTextValidate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    any
		wantErr error
	}{
		{name: "struct", have: &texttest.Handle{}},
		{name: "enum", have: texttest.Type_Company},
		{name: "plain enum", have: texttest.Plain_FooBar},
		{name: "no Format", have: &texttest.Unannotated{}, wantErr: text.ErrInvalidDefinition},
		{name: "optional field", have: &texttest.OptionalField{}, wantErr: text.ErrInvalidDefinition},
		{name: "union", have: &texttest.Union{}, wantErr: text.ErrInvalidDefinition},
		{
			name:    "separator conflict",
			have:    &texttest.SeparatorConflict{},
			wantErr: text.ErrInvalidDefinition,
		},
		{name: "struct value", have: texttest.Handle{}, wantErr: text.ErrInvalidDefinition},
		{name: "not thrift", have: 42, wantErr: text.ErrInvalidDefinition},
		{name: "nil", have: nil, wantErr: text.ErrInvalidDefinition},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, text.Validate(tt.have), tt.wantErr)
		})
	}
}

func TestTextNestedStruct(t *testing.T) {
	for _, tt := range []struct {
		name    string
		have    thrift.RegistrableStruct
		want    string
		wantErr error
	}{
		{
			name: "embedded struct",
			have: &texttest.Labeled{
				Owner: &texttest.Ownership{Type: texttest.Type_Team, EntityID: 3},
				Label: "a/b",
			},
			want: "team:3/a/b",
		},
		{
			name: "embedded twice",
			have: &texttest.Wrapped{
				Inner: &texttest.Labeled{
					Owner: &texttest.Ownership{Type: texttest.Type_GoogleOAuth, EntityID: 1},
					Label: "x",
				},
			},
			want: "<gmail:1/x>",
		},
		{
			name:    "missing embedded struct",
			have:    &texttest.Labeled{Label: "x"},
			wantErr: text.ErrInvalidValue,
		},
		{
			name:    "embedded struct without Format",
			have:    &texttest.NestedUnannotated{},
			wantErr: text.ErrInvalidDefinition,
		},
		{
			name:    "struct embedding itself",
			have:    &texttest.Loop{},
			wantErr: text.ErrInvalidDefinition,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := text.MarshalStruct(tt.have)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}

	got, err := text.UnmarshalStruct[texttest.Wrapped]("<Type_Company:7/a:b>")

	require.NoError(t, err)
	assert.Equal(
		t,
		&texttest.Wrapped{
			Inner: &texttest.Labeled{
				Owner: &texttest.Ownership{Type: texttest.Type_Company, EntityID: 7},
				Label: "a:b",
			},
		},
		got,
	)
}

func TestTextOmittedOptionalField(t *testing.T) {
	handle := "bob"

	s, err := text.MarshalStruct(
		&texttest.WithHandle{Media: texttest.MediaType_MediaTiktok, ID: 2, Handle: &handle},
	)

	require.NoError(t, err)
	assert.Equal(t, "tt:2", s)

	got, err := text.UnmarshalStruct[texttest.WithHandle]("youtube:3")

	require.NoError(t, err)
	assert.Equal(t, &texttest.WithHandle{Media: texttest.MediaType_MediaYoutubeShort, ID: 3}, got)

	node, err := text.UnmarshalStruct[texttest.Node]("4")

	require.NoError(t, err)
	assert.Equal(t, &texttest.Node{ID: 4}, node)
}
