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

THRIFTCMD = $(THRIFT) -out gen --gen go$(COMPILER_EXTRAFLAG)
THRIFTTEST = $(top_srcdir)/test/ThriftTest.thrift
STREAMINGTEST = $(top_srcdir)/test/StreamingTest.thrift

ThriftTest.thrift: $(THRIFTTEST)
	grep -v list.*map.*list.*map $(THRIFTTEST) > ThriftTest.thrift

gopath.stamp: $(THRIFT) ThriftTest.thrift
	mkdir -p gen
	$(THRIFTCMD) ThriftTest.thrift
	$(THRIFTCMD) ../StressTest.thrift
	$(THRIFTCMD) $(STREAMINGTEST)
	test -f $(top_srcdir)/go.mod || printf 'module github.com/upfluence/thrift\n\ngo $(golang_version)\n' > $(top_srcdir)/go.mod
	$(GO) mod tidy
	touch gopath.stamp

genmock: gopath.stamp
	$(GO) install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=gen/thrifttest/ThriftTest.go \
		-destination=src/common/mock_handler.go \
		-package=common

check: gopath.stamp genmock
	$(GO) test -v ./src/common/...

clean-local:
	$(RM) -r gen gopath.stamp bin ThriftTest.thrift go.sum $(top_srcdir)/go.mod $(top_srcdir)/go.sum

EXTRA_DIST = \
	src/bin \
	src/common
