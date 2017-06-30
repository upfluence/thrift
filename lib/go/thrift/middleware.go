package thrift

type TResponse interface {
	TStruct
	IsSetSuccess() bool
	GetError() error
}

type TRequest interface {
	TStruct
}

type TBinaryTransaction interface {
	Handle(TResponse, error) error
}

type TUnaryTransaction interface {
	Handle(error) error
}

type TMiddleware interface {
	HandleBinaryRequest(string, int32, TRequest) (TBinaryTransaction, error)
	HandleUnaryRequest(string, int32, TRequest) (TUnaryTransaction, error)
}

type TMiddlewareBuilder interface {
	Build(string) TMiddleware
}
