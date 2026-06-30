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

THRIFTARGS = -out gen --gen go:package_prefix=github.com/upfluence/thrift/lib/go/test/gen/$(COMPILER_EXTRAFLAG)
THRIFTTEST = $(top_srcdir)/test/ThriftTest.thrift

# Thrift for GO has problems with complex map keys: THRIFT-2063
gopath: $(THRIFT) $(THRIFTTEST) \
			IncludesTest.thrift \
			NamespacedTest.thrift \
			MultiplexedProtocolTest.thrift \
			OnewayTest.thrift \
			OptionalFieldsTest.thrift \
			RequiredFieldTest.thrift \
			ServicesTest.thrift \
			GoTagTest.thrift \
			TypedefFieldTest.thrift \
			RefAnnotationFieldsTest.thrift \
			UnionDefaultValueTest.thrift \
			UnionBinaryTest.thrift \
			ErrorTest.thrift \
			NamesTest.thrift \
			InitialismsTest.thrift \
			DontExportRWTest.thrift \
			IgnoreInitialismsTest.thrift \
			ConflictNamespaceTestA.thrift \
			ConflictNamespaceTestB.thrift \
			ConflictNamespaceTestC.thrift \
			ConflictNamespaceTestD.thrift \
			ConflictNamespaceTestSuperThing.thrift \
			ConflictNamespaceServiceTest.thrift
	mkdir -p gen
	grep -v list.*map.*list.*map $(THRIFTTEST) | grep -v 'set<Insanity>' > ThriftTest.thrift
	$(THRIFT) $(THRIFTARGS) -r IncludesTest.thrift
	$(THRIFT) $(THRIFTARGS) BinaryKeyTest.thrift
	$(THRIFT) $(THRIFTARGS) MultiplexedProtocolTest.thrift
	$(THRIFT) $(THRIFTARGS) OnewayTest.thrift
	$(THRIFT) $(THRIFTARGS) OptionalFieldsTest.thrift
	$(THRIFT) $(THRIFTARGS) RequiredFieldTest.thrift
	$(THRIFT) $(THRIFTARGS) ServicesTest.thrift
	$(THRIFT) $(THRIFTARGS) GoTagTest.thrift
	$(THRIFT) $(THRIFTARGS) TypedefFieldTest.thrift
	$(THRIFT) $(THRIFTARGS) RefAnnotationFieldsTest.thrift
	$(THRIFT) $(THRIFTARGS) UnionDefaultValueTest.thrift
	$(THRIFT) $(THRIFTARGS) UnionBinaryTest.thrift
	$(THRIFT) $(THRIFTARGS) ErrorTest.thrift
	$(THRIFT) $(THRIFTARGS) NamesTest.thrift
	$(THRIFT) $(THRIFTARGS) InitialismsTest.thrift
	$(THRIFT) $(THRIFTARGS),read_write_private DontExportRWTest.thrift
	$(THRIFT) $(THRIFTARGS),ignore_initialisms IgnoreInitialismsTest.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceTestA.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceTestB.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceTestC.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceTestD.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceTestSuperThing.thrift
	$(THRIFT) $(THRIFTARGS) ConflictNamespaceServiceTest.thrift
	cp -r dontexportrwtest gen/
	$(GO) mod tidy
	touch gopath

check: gopath
	$(GO) test -vet=off ./tests/... ./gen/dontexportrwtest/...

.PHONY: gopath

clean-local:
	$(RM) -r gen ThriftTest.thrift gen-go

EXTRA_DIST = \
	dontexportrwtest \
	tests \
	BinaryKeyTest.thrift \
	GoTagTest.thrift \
	IncludesTest.thrift \
	MultiplexedProtocolTest.thrift \
	NamespacedTest.thrift \
	OnewayTest.thrift \
	OptionalFieldsTest.thrift \
	RequiredFieldTest.thrift \
	RefAnnotationFieldsTest.thrift \
	UnionDefaultValueTest.thrift \
	UnionBinaryTest.thrift \
	ServicesTest.thrift \
	TypedefFieldTest.thrift \
	ErrorTest.thrift \
	NamesTest.thrift \
	InitialismsTest.thrift \
	DontExportRWTest.thrift \
	IgnoreInitialismsTest.thrift \
	ConflictNamespaceTestA.thrift \
	ConflictNamespaceTestB.thrift \
	ConflictNamespaceTestC.thrift \
	ConflictNamespaceTestD.thrift \
	ConflictNamespaceTestSuperThing.thrift \
	ConflictNamespaceServiceTest.thrift
