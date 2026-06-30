module github.com/upfluence/thrift/tutorial/go

go 1.26.2

replace github.com/upfluence/thrift => ../..

require (
	github.com/apache/thrift v0.23.0
	github.com/upfluence/thrift v0.0.0-00010101000000-000000000000
)

require github.com/upfluence/errors v0.2.19 // indirect
