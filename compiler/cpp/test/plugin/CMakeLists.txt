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

find_program(GO_EXECUTABLE go)

if(GO_EXECUTABLE)
    # Write a minimal root go.mod so that the replace directive in
    # echo_plugin/go.mod (replace github.com/upfluence/thrift => ../../../../..)
    # can be resolved by `go build`.  The file is not committed to the repo;
    # it is generated on demand and the path is gitignored.
    set(ROOT_GO_MOD ${CMAKE_SOURCE_DIR}/go.mod)
    add_custom_command(
        OUTPUT ${ROOT_GO_MOD}
        COMMAND ${CMAKE_COMMAND} -E echo "module github.com/upfluence/thrift" > ${ROOT_GO_MOD}
        COMMAND ${CMAKE_COMMAND} -E echo "" >> ${ROOT_GO_MOD}
        COMMAND ${CMAKE_COMMAND} -E echo "go 1.25.6" >> ${ROOT_GO_MOD}
        COMMENT "Generating root go.mod"
    )

    # Build the echo plugin from its own module (uses a replace directive to
    # point at the local lib/go/thrift sources).
    add_custom_command(
        OUTPUT ${CMAKE_CURRENT_BINARY_DIR}/echo_plugin
        COMMAND ${GO_EXECUTABLE} build -o ${CMAKE_CURRENT_BINARY_DIR}/echo_plugin .
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/echo_plugin
        DEPENDS ${CMAKE_CURRENT_SOURCE_DIR}/echo_plugin/main.go
                ${CMAKE_CURRENT_SOURCE_DIR}/echo_plugin/go.mod
                ${ROOT_GO_MOD}
        COMMENT "Building echo_plugin"
    )
    add_custom_target(build_echo_plugin ALL
        DEPENDS ${CMAKE_CURRENT_BINARY_DIR}/echo_plugin
    )
    add_dependencies(thrift-compiler build_echo_plugin)

    # Test 1: the compiler invokes the plugin and produces the output file.
    # We first create the output directory via a fixture setup step, then run
    # the compiler. The plugin output directory (gen-echo/) inside it is
    # created by the compiler via MKDIR.
    add_test(NAME PluginOutputDirSetup
        COMMAND ${CMAKE_COMMAND} -E make_directory
            ${CMAKE_CURRENT_BINARY_DIR}/plugin-out
    )
    set_tests_properties(PluginOutputDirSetup PROPERTIES
        FIXTURES_SETUP PluginOutputDir
    )

    add_test(NAME PluginOutputFileTest
        COMMAND thrift-compiler
            --gen go
            --out ${CMAKE_CURRENT_BINARY_DIR}/plugin-out
            -plugin echo=${CMAKE_CURRENT_BINARY_DIR}/echo_plugin
            -plugin-out echo=gen-echo
            ${CMAKE_CURRENT_SOURCE_DIR}/input.thrift
        WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
    )
    set_tests_properties(PluginOutputFileTest PROPERTIES
        FIXTURES_REQUIRED PluginOutputDir
    )

    # Test 2: the generated file matches the committed golden output.
    add_test(NAME PluginGoldenTest
        COMMAND ${CMAKE_COMMAND} -E compare_files
            ${CMAKE_CURRENT_BINARY_DIR}/plugin-out/gen-echo/echo.txt
            ${CMAKE_CURRENT_SOURCE_DIR}/golden.txt
    )
    set_tests_properties(PluginGoldenTest PROPERTIES
        FIXTURES_REQUIRED PluginOutputDir
        DEPENDS PluginOutputFileTest
    )

    # Test 3: -plugin-out echo=. writes the file directly into the --out
    # directory, alongside the generator's own output (no subdirectory).
    add_test(NAME PluginDotOutDirSetup
        COMMAND ${CMAKE_COMMAND} -E make_directory
            ${CMAKE_CURRENT_BINARY_DIR}/plugin-out-dot
    )
    set_tests_properties(PluginDotOutDirSetup PROPERTIES
        FIXTURES_SETUP PluginDotOutDir
    )

    add_test(NAME PluginDotOutFileTest
        COMMAND thrift-compiler
            --gen go
            --out ${CMAKE_CURRENT_BINARY_DIR}/plugin-out-dot
            -plugin echo=${CMAKE_CURRENT_BINARY_DIR}/echo_plugin
            -plugin-out echo=.
            ${CMAKE_CURRENT_SOURCE_DIR}/input.thrift
        WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
    )
    set_tests_properties(PluginDotOutFileTest PROPERTIES
        FIXTURES_REQUIRED PluginDotOutDir
    )

    # The echo plugin produces the same content regardless of output path;
    # reuse golden.txt for the comparison.
    add_test(NAME PluginDotOutGoldenTest
        COMMAND ${CMAKE_COMMAND} -E compare_files
            ${CMAKE_CURRENT_BINARY_DIR}/plugin-out-dot/echo.txt
            ${CMAKE_CURRENT_SOURCE_DIR}/golden.txt
    )
    set_tests_properties(PluginDotOutGoldenTest PROPERTIES
        FIXTURES_REQUIRED PluginDotOutDir
        DEPENDS PluginDotOutFileTest
    )
else()
    message(WARNING "Go not found; skipping plugin integration tests")
endif()
