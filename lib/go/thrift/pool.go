package thrift

type TPoolProvider interface {
	Build(TClientProvider) (TPool, error)
}

type TPool interface {
	Get() (interface{}, error)
	Put(interface{}) error
}
