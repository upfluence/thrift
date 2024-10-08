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

cmake_minimum_required(VERSION 2.8.12)

project(thrift)

set(TYPES_PREFIX "/usr/local/share/thrift" CACHE STRING "Directory the thrift tpyes will be installed in")
message(STATUS "Types will be installed in ${TYPES_PREFIX}")

file(COPY "${CMAKE_CURRENT_SOURCE_DIR}/types" DESTINATION "${TYPES_PREFIX}")

set(CMAKE_MODULE_PATH "${CMAKE_MODULE_PATH}" "${CMAKE_CURRENT_SOURCE_DIR}/build/cmake")

# TODO: add `git rev-parse --short HEAD`
# Read the version information from the Autoconf file
file (STRINGS "${CMAKE_CURRENT_SOURCE_DIR}/configure.ac" CONFIGURE_AC REGEX "AC_INIT\\(.*\\)" )
option (THRIFT_VERSION "VERSION" ${CONFIGURE_AC})

# The following variable is used in the version.h.in file
string(REGEX REPLACE "AC_INIT\\(\\[.*\\], \\[([0-9]+\\.[0-9]+\\.[0-9]+(-upfluence)?)\\]\\)" "\\1" PACKAGE_VERSION ${THRIFT_VERSION})
message(STATUS "Parsed Thrift package version: ${PACKAGE_VERSION}")

set(REPO_VERSION "${THRIFT_VERSION}")

# These are internal to CMake
string(REGEX REPLACE "([0-9]+\\.[0-9]+\\.[0-9]+)(-upfluence)?" "\\1" thrift_VERSION ${PACKAGE_VERSION})
string(REGEX REPLACE "([0-9]+)\\.[0-9]+\\.[0-9]+" "\\1" thrift_VERSION_MAJOR ${thrift_VERSION})
string(REGEX REPLACE "[0-9]+\\.([0-9])+\\.[0-9]+" "\\1" thrift_VERSION_MINOR ${thrift_VERSION})
string(REGEX REPLACE "[0-9]+\\.[0-9]+\\.([0-9]+)" "\\1" thrift_VERSION_PATCH ${thrift_VERSION})
message(STATUS "Parsed Thrift version: ${thrift_VERSION} (${thrift_VERSION_MAJOR}.${thrift_VERSION_MINOR}.${thrift_VERSION_PATCH})")

# Some default settings
include(DefineCMakeDefaults)

# Build time options are defined here
include(DefineOptions)
include(DefineInstallationPaths)

# Based on the options set some platform specifics
include(DefinePlatformSpecifc)

# Generate the config.h file
include(ConfigureChecks)

# Package it
include(CPackConfig)


find_package(Threads)

include(CTest)
if(BUILD_TESTING)
  message(STATUS "Building with unittests")

  enable_testing()
  # Define "make check" as alias for "make test"
  add_custom_target(check COMMAND ctest)
else ()
  message(STATUS "Building without tests")
endif ()
if(BUILD_COMPILER)
    add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/compiler/cpp)
endif()

if(BUILD_CPP)
    add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/lib/cpp)
    if(BUILD_TESTING)
        if(WITH_LIBEVENT AND WITH_ZLIB AND WITH_OPENSSL)
            add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/test/cpp)
        else()
            message(WARNING "libevent and/or ZLIB and/or OpenSSL not found or disabled; will not build some tests")
        endif()
    endif()
endif()

if(BUILD_C_GLIB)
    add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/lib/c_glib)
endif()

if(BUILD_JAVA)
    add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/lib/java)
endif()

if(BUILD_PYTHON)
    add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/lib/py)
endif()

PRINT_CONFIG_SUMMARY()
