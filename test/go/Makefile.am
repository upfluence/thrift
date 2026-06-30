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

ThriftTest.thrift: $(THRIFTTEST)
	grep -v list.*map.*list.*map $(THRIFTTEST) > ThriftTest.thrift

gopath: $(THRIFT) ThriftTest.thrift
	mkdir -p gen
	$(THRIFTCMD) ThriftTest.thrift
	$(THRIFTCMD) ../StressTest.thrift
	$(GO) mod tidy
	touch gen.stamp

genmock: gopath
	$(GO) install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=gen/thrifttest/ThriftTest.go \
		-destination=src/common/mock_handler.go \
		-package=common

check: gopath genmock
	$(GO) test -v ./src/common/...

clean-local:
	$(RM) -r gen gen.stamp bin ThriftTest.thrift go.sum

EXTRA_DIST = \
	src/bin \
	src/common
