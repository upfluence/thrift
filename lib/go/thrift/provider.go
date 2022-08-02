package thrift

type TClientProvider interface {
	Build(string, string) (TClient, error)
}

type TStreamingClientProvider interface {
	BuildStreaming(string, string) (TStreamingClient, error)
}

type TProcessorProvider interface {
	Build(string, string) (TProcessor, error)
}
