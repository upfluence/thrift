package thrift

type TResponse interface {
	TStruct
	GetResult() interface{}
	GetError() error
}

type TRequest interface {
	TStruct
}

type TMiddleware interface {
	HandleBinaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error)
	HandleUnaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) error) error
}

type TStreamingMiddleware interface {
	TMiddleware

	HandleInboundStream(ctx Context, mth string, seqID int32, req TRequest, s TInboundStream, next func(Context, TRequest, TInboundStream) (TResponse, error)) (TResponse, error)
	HandleOutboundStream(ctx Context, mth string, seqID int32, req TRequest, s TOutboundStream, next func(Context, TRequest, TOutboundStream) (TResponse, error)) (TResponse, error)
	HandleBidiStream(ctx Context, mth string, seqID int32, req TRequest, is TInboundStream, os TOutboundStream, next func(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)) (TResponse, error)
}

type TMiddlewareBuilder interface {
	Build(namespace, service string) TMiddleware
}

type TNopMiddleware struct{}

func (TNopMiddleware) HandleBinaryRequest(ctx Context, _ string, _ int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	return next(ctx, req)
}

func (TNopMiddleware) HandleUnaryRequest(ctx Context, _ string, _ int32, req TRequest, next func(Context, TRequest) error) error {
	return next(ctx, req)
}

func (TNopMiddleware) HandleInboundStream(ctx Context, _ string, _ int32, req TRequest, s TInboundStream, next func(Context, TRequest, TInboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, s)
}

func (TNopMiddleware) HandleOutboundStream(ctx Context, _ string, _ int32, req TRequest, s TOutboundStream, next func(Context, TRequest, TOutboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, s)
}

func (TNopMiddleware) HandleBidiStream(ctx Context, _ string, _ int32, req TRequest, is TInboundStream, os TOutboundStream, next func(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, is, os)
}

type TMultiMiddleware []TStreamingMiddleware

func (ms TMultiMiddleware) HandleBinaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) (TResponse, error)) (TResponse, error) {
	for i := len(ms); i > 0; i-- {
		call := next
		i := i
		next = func(ctx Context, req TRequest) (TResponse, error) {
			return ms[i-1].HandleBinaryRequest(ctx, mth, seqID, req, call)
		}
	}

	return next(ctx, req)
}

func (ms TMultiMiddleware) HandleUnaryRequest(ctx Context, mth string, seqID int32, req TRequest, next func(Context, TRequest) error) error {
	for i := len(ms); i > 0; i-- {
		call := next
		i := i
		next = func(ctx Context, req TRequest) error {
			return ms[i-1].HandleUnaryRequest(ctx, mth, seqID, req, call)
		}
	}

	return next(ctx, req)
}

func (ms TMultiMiddleware) HandleInboundStream(ctx Context, mth string, seqID int32, req TRequest, s TInboundStream, next func(Context, TRequest, TInboundStream) (TResponse, error)) (TResponse, error) {
	for i := len(ms); i > 0; i-- {
		call := next
		i := i
		next = func(ctx Context, req TRequest, s TInboundStream) (TResponse, error) {
			return ms[i-1].HandleInboundStream(ctx, mth, seqID, req, s, call)
		}
	}

	return next(ctx, req, s)
}

func (ms TMultiMiddleware) HandleOutboundStream(ctx Context, mth string, seqID int32, req TRequest, s TOutboundStream, next func(Context, TRequest, TOutboundStream) (TResponse, error)) (TResponse, error) {
	for i := len(ms); i > 0; i-- {
		call := next
		i := i
		next = func(ctx Context, req TRequest, s TOutboundStream) (TResponse, error) {
			return ms[i-1].HandleOutboundStream(ctx, mth, seqID, req, s, call)
		}
	}

	return next(ctx, req, s)
}

func (ms TMultiMiddleware) HandleBidiStream(ctx Context, mth string, seqID int32, req TRequest, is TInboundStream, os TOutboundStream, next func(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)) (TResponse, error) {
	for i := len(ms); i > 0; i-- {
		call := next
		i := i
		next = func(ctx Context, req TRequest, is TInboundStream, os TOutboundStream) (TResponse, error) {
			return ms[i-1].HandleBidiStream(ctx, mth, seqID, req, is, os, call)
		}
	}

	return next(ctx, req, is, os)
}

type upgradedMiddleware struct {
	TMiddleware
}

func (upgradedMiddleware) HandleInboundStream(ctx Context, _ string, _ int32, req TRequest, s TInboundStream, next func(Context, TRequest, TInboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, s)
}

func (upgradedMiddleware) HandleOutboundStream(ctx Context, _ string, _ int32, req TRequest, s TOutboundStream, next func(Context, TRequest, TOutboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, s)
}

func (upgradedMiddleware) HandleBidiStream(ctx Context, _ string, _ int32, req TRequest, is TInboundStream, os TOutboundStream, next func(Context, TRequest, TInboundStream, TOutboundStream) (TResponse, error)) (TResponse, error) {
	return next(ctx, req, is, os)
}

func upgradeMiddleware(m TMiddleware) TStreamingMiddleware {
	if sm, ok := m.(TStreamingMiddleware); ok {
		return sm
	}

	return upgradedMiddleware{TMiddleware: m}
}

func WrapMiddlewares(ms []TMiddleware) TStreamingMiddleware {
	switch len(ms) {
	case 0:
		return TNopMiddleware{}
	case 1:
		return upgradeMiddleware(ms[0])
	}

	sms := make([]TStreamingMiddleware, len(ms))

	for i, m := range ms {
		sms[i] = upgradeMiddleware(m)
	}

	return TMultiMiddleware(sms)
}
