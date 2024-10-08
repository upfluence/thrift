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

DESTDIR ?= /
THRIFT = $(top_builddir)/compiler/cpp/thrift

if HAVE_BUNDLER

all-local:
	$(BUNDLER) install
	$(BUNDLER) exec rake build_ext

install-exec-hook:
	$(BUNDLER) exec rake install

clean-local:
	$(BUNDLER) exec rake clean

check-local: all
	$(BUNDLER) install
	$(BUNDLER) exec rake

endif

TYPE_FILES = ../../types/*.thrift ../../types/*/*.thrift

$(TYPE_FILES):
	mkdir -p lib/thrift/$(patsubst ../../%,%,$(dir $@))
	$(THRIFT) -gen rb:namespaced,namespace_wrapper=thrift \
		     --out lib/ $@

generate-types: $(TYPE_FILES)

.PHONY: generate-types $(TYPE_FILES)

EXTRA_DIST = \
	Rakefile \
	Gemfile \
	thrift.gemspec \
	lib \
	ext \
	benchmark \
	script \
	spec \
	README.md
