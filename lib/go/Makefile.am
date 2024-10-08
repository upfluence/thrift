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

SUBDIRS = .

if WITH_TESTS
SUBDIRS += test
endif

install:
	@echo '##############################################################'
	@echo '##############################################################'
	@echo 'The Go client library should be installed via "go get", please see /lib/go/README.md'
	@echo '##############################################################'
	@echo '##############################################################'

TYPE_FILES = ../../types/*.thrift ../../types/*/*.thrift

$(TYPE_FILES):
	$(THRIFT) -gen go --out thrift $@

generate-types: $(TYPE_FILES)

.PHONY: generate-types $(TYPE_FILES)

check-local:
	$(GO) test ./thrift

all-local:
	$(GO) build ./thrift

EXTRA_DIST = \
	thrift \
	README.md
