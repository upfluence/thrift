#!/bin/bash

set -ex

git_tag=$(git describe --tag)
tag=${TAG:-$git_tag}
version="${tag%-upfluence}-upfluence"

thrift_bin="$(pwd)/bin/thrift"
if ! "$thrift_bin" --version &>/dev/null; then
  thrift_bin="$(which thrift)"
fi

build_dir="$(mktemp -d)"
trap 'rm -rf "$build_dir"' EXIT

cmake -S . -B "$build_dir" \
  -DTHRIFT_VERSION="$version" \
  -DTHRIFT_COMPILER="$thrift_bin" \
  -DBUILD_LIBRARIES=OFF \
  -DBUILD_TESTING=OFF

cmake --build "$build_dir" --target generate-types
