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

set(THRIFT_TYPES_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../types")
find_program(THRIFT_COMPILER_PATH thrift)
if(THRIFT_COMPILER_PATH)
    set(THRIFT_COMPILER_EXECUTABLE "${THRIFT_COMPILER_PATH}")
else()
    set(THRIFT_COMPILER_EXECUTABLE "${THRIFT_COMPILER}")
endif()
file(GLOB_RECURSE THRIFT_TYPE_FILES "${THRIFT_TYPES_DIR}/*.thrift")
set(THRIFT_TYPE_STAMPS "")
foreach(THRIFT_TYPE_FILE ${THRIFT_TYPE_FILES})
    file(RELATIVE_PATH THRIFT_TYPE_RELATIVE "${THRIFT_TYPES_DIR}" "${THRIFT_TYPE_FILE}")
    get_filename_component(THRIFT_TYPE_RELATIVE_DIR "${THRIFT_TYPE_RELATIVE}" DIRECTORY)
    set(THRIFT_TYPE_STAMP "${CMAKE_CURRENT_BINARY_DIR}/generated-types/${THRIFT_TYPE_RELATIVE}.stamp")
    get_filename_component(THRIFT_TYPE_STAMP_DIR "${THRIFT_TYPE_STAMP}" DIRECTORY)
    if(THRIFT_TYPE_RELATIVE_DIR)
        set(THRIFT_TYPE_OUT_DIR "${CMAKE_CURRENT_SOURCE_DIR}/lib/thrift/${THRIFT_TYPE_RELATIVE_DIR}")
    else()
        set(THRIFT_TYPE_OUT_DIR "${CMAKE_CURRENT_SOURCE_DIR}/lib/thrift")
    endif()
    add_custom_command(
        OUTPUT "${THRIFT_TYPE_STAMP}"
        COMMAND ${CMAKE_COMMAND} -E make_directory "${THRIFT_TYPE_OUT_DIR}"
        COMMAND ${CMAKE_COMMAND} -E make_directory "${THRIFT_TYPE_STAMP_DIR}"
        COMMAND ${THRIFT_COMPILER_EXECUTABLE} -gen rb:namespaced,namespace_wrapper=thrift --out "${CMAKE_CURRENT_SOURCE_DIR}/lib" "${THRIFT_TYPE_FILE}"
        COMMAND ${CMAKE_COMMAND} -E touch "${THRIFT_TYPE_STAMP}"
        DEPENDS "${THRIFT_TYPE_FILE}"
        COMMENT "Generating Ruby thrift types from ${THRIFT_TYPE_RELATIVE}"
    )
    list(APPEND THRIFT_TYPE_STAMPS "${THRIFT_TYPE_STAMP}")
endforeach()

add_custom_target(generate-types-rb DEPENDS ${THRIFT_TYPE_STAMPS})
