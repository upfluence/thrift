module github.com/upfluence/thrift/test/go

go 1.23.0

replace github.com/upfluence/thrift => ../../

require (
	github.com/golang/mock v1.6.0
	github.com/upfluence/thrift v0.0.0-00010101000000-000000000000
)

require github.com/upfluence/errors v0.2.11 // indirect
