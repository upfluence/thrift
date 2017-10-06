package thrift

type TClientProvider interface {
	Build(string, string) (TTransport, TProtocolFactory, TMiddleware, error)
}
