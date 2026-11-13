package naming

import (
	"github.com/upfluence/thrift/lib/go/thrift"
	"github.com/upfluence/thrift/lib/go/thrift/internal/reflection"
)

func init() {
	reflection.RegisterCanonicalNameExtractor(
		reflection.CanonicalNameExtractorFunc(func(ns string, def reflection.AnnotatedDefinition) []string {
			var res []string

			for _, pka := range thrift.FindAnnotations[*PreviouslyKnownAs](def) {
				lns := pka.GetNamespace_()
				ln := pka.GetName()

				if pka.Namespace_ == nil {
					lns = ns
				}

				if ln == "" {
					ln = def.Name
				}

				res = append(res, lns+"."+ln)
			}

			return res
		}),
	)
}
