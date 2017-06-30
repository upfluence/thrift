package thrift

type TClientProvider interface {
	Build(string) (TTransport, TProtocolFactory, TMiddleware, error)
}

type TServerProvider interface {
	Build(string) (TServerFactory, TProtocolFactory, TMiddleware, error)
}
