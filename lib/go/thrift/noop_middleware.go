package thrift

var DefaultMiddlewareBuilder = &TNoopMiddlewareBuilder{}

type TNoopMiddlewareBuilder struct{}
type TNoopMiddleware struct{}
type noopUnaryTransaction struct{}
type noopBinaryTransaction struct{}

func (b *TNoopMiddlewareBuilder) Build(_, _ string) TMiddleware {
	return &TNoopMiddleware{}
}

func (m *TNoopMiddleware) HandleUnaryRequest(_ string, _ int32, _ TRequest) (TUnaryTransaction, error) {
	return &noopUnaryTransaction{}, nil
}

func (b *TNoopMiddleware) HandleBinaryRequest(_ string, _ int32, _ TRequest) (TBinaryTransaction, error) {
	return &noopBinaryTransaction{}, nil
}

func (t *noopBinaryTransaction) Handle(_ TResponse, _ error) error {
	return nil
}

func (t *noopUnaryTransaction) Handle(_ error) error {
	return nil
}
