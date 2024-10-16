module github.com/upfluence/thrift/lib/go/test

go 1.23.0

replace github.com/upfluence/thrift => ../../..

require (
	github.com/golang/mock v1.6.0
	github.com/upfluence/thrift v2.6.5+incompatible
)

require github.com/upfluence/errors v0.2.11 // indirect
