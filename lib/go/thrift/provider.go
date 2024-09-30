package thrift

import "errors"

var (
	ErrClientNotDefined    = errors.New("client not defined")
	ErrProcessorNotDefined = errors.New("processor not defined")
)

type TClientProvider interface {
	Build(string, string) (TClient, error)
}

func BuildClient(cp TClientProvider, s FullyDefinedStruct) (TClient, error) {
	defs := append(
		[]StructDefinition{s.StructDefinition()},
		s.LegacyStructDefinitions()...,
	)

	for _, def := range defs {
		switch cl, err := cp.Build(def.Namespace, def.Name); {
		case err == nil:
			return cl, nil
		case errors.Is(err, ErrClientNotDefined):
		default:
			return nil, err
		}
	}

	return nil, ErrClientNotDefined
}

type TProcessorProvider interface {
	Build(string, string) (TProcessor, error)
}

func BuildProcessor(pp TProcessorProvider, s FullyDefinedStruct) (TProcessor, error) {
	defs := append(
		[]StructDefinition{s.StructDefinition()},
		s.LegacyStructDefinitions()...,
	)

	for _, def := range defs {
		switch cl, err := pp.Build(def.Namespace, def.Name); {
		case err == nil:
			return cl, nil
		case errors.Is(err, ErrProcessorNotDefined):
		default:
			return nil, err
		}
	}

	return nil, ErrProcessorNotDefined
}
