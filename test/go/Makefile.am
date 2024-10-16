#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements. See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership. The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.
#

THRIFT = $(top_builddir)/compiler/cpp/thrift
THRIFTARGS = -out src/gen --gen go:package_prefix=github.com/upfluence/thrift/test/go/src/gen/
THRIFTTEST = $(top_srcdir)/test/ThriftTest.thrift
THRIFTCMD = $(THRIFT) $(THRIFTARGS)

precross: bin/testclient bin/testserver

ThriftTest.thrift: $(THRIFTTEST)
	grep -v list.*map.*list.*map $(THRIFTTEST) > ThriftTest.thrift

# Thrift for GO has problems with complex map keys: THRIFT-2063
gopath: $(THRIFT) ThriftTest.thrift
	mkdir -p src/gen
	$(THRIFTCMD) ThriftTest.thrift
	$(THRIFTCMD) ../StressTest.thrift
	$(GO) get github.com/golang/mock/gomock

bin/testclient: gopath
	$(GO) install ./src/bin/testclient

bin/testserver: gopath
	$(GO) install ./src/bin/testserver

bin/stress: gopath
	$(GO) install ./src/bin/stress

clean-local:
	$(RM) -r src/gen src/code.google.com src/thrift bin pkg gopath ThriftTest.thrift

check_PROGRAMS: bin/testclient bin/testserver bin/stress

check: gopath
	$(GO) test -v ./src/common/...
	$(GO) test -v ./src/gen/...

genmock: gopath
	bin/mockgen -destination=src/common/mock_handler.go -package=common gen/thrifttest ThriftTest

EXTRA_DIST = \
	src/bin \
	src/common
