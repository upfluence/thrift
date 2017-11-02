package thrift

var DefaultMiddlewareBuilder = &TNoopMiddlewareBuilder{}

type TNoopMiddlewareBuilder struct{}
type TNoopMiddleware struct{}
type noopUnaryTransaction struct{}
type noopBinaryTransaction struct{}

func (b *TNoopMiddlewareBuilder) Build(_, _ string) TMiddleware {
	return &TNoopMiddleware{}
}

func (m *TNoopMiddleware) HandleUnaryRequest(ctx Context, _ string, _ int32, req TRequest, next func(Context, TRequest) error) error {
	return next(ctx, req)
}

func (b *TNoopMiddleware) HandleBinaryRequest(ctx Context, _ string, _ int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	return next(ctx, req)
}
