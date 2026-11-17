package thrift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fooAnnotation struct{ v int }

func (*fooAnnotation) StructDefinition() StructDefinition { return StructDefinition{} }

type barAnnotation struct{}

func (*barAnnotation) StructDefinition() StructDefinition { return StructDefinition{} }

func TestFindAnnotations(t *testing.T) {
	var (
		foo1 = &fooAnnotation{v: 1}
		foo2 = &fooAnnotation{v: 2}
		bar  = &barAnnotation{}
	)

	for _, tt := range []struct {
		name string
		have []RegistrableStruct
		want []*fooAnnotation
	}{
		{name: "no annotation"},
		{name: "other annotations", have: []RegistrableStruct{bar}},
		{
			name: "declaration order",
			have: []RegistrableStruct{foo2, bar, foo1},
			want: []*fooAnnotation{foo2, foo1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := FindAnnotations[*fooAnnotation](
				AnnotatedDefinition{StructuredAnnotations: tt.have},
			)

			assert.Equal(t, tt.want, got)
		})
	}
}
