package thrift

type TClientProvider interface {
	Build(string, string) (TTransport, TProtocolFactory, TMiddleware, error)
}

type TServerProvider interface {
	Build(string, string) (TServerFactory, TProtocolFactory, TMiddleware, error)
}
