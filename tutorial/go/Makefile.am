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

THRIFTARGS = --gen go:package_prefix=github.com/upfluence/thrift/tutorial/go/gen-go/$(COMPILER_EXTRAFLAG)
gen-go/tutorial/calculator.go gen-go/shared/shared_service.go: $(top_srcdir)/tutorial/tutorial.thrift
	$(THRIFT) $(THRIFTARGS) -r $<

all-local: gen-go/tutorial/calculator.go

check: src/github.com/apache/thrift/lib/go/thrift thirdparty-dep
	$(THRIFT) -r $(THRIFTARGS) $(top_srcdir)/tutorial/tutorial.thrift
	$(GO) test ./...

thirdparty-dep:

tutorialserver: all
	$(GO) run src/*.go -server=true

tutorialclient: all
	$(GO) run src/*.go

tutorialsecureserver: all
	$(GO) run src/*.go -server=true -secure=true

tutorialsecureclient: all
	$(GO) run src/*.go -secure=true

clean-local:
	$(RM) -r gen-*

EXTRA_DIST = \
	src/client.go \
	src/handler.go \
	src/server.go \
	src/main.go \
	server.crt \
	server.key

