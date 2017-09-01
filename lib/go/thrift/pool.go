package thrift

type TPoolProvider interface {
	BuildClient() TClientProvider
	BuildPool(func() (interface{}, error)) (TPool, error)
}

type TPool interface {
	Get() (interface{}, error)
	Put(interface{}) error
	Discard(interface{}) error
}
