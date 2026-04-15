#!/bin/bash

set -ex

git_tag=$(git describe --tag)
tag=${TAG:-$git_tag}
version="${tag#v}-upfluence"

build_dir="$(mktemp -d)"
trap 'rm -rf "$build_dir"' EXIT

cmake -S . -B "$build_dir" \
  -DTHRIFT_VERSION="$version" \
  -DBUILD_LIBRARIES=OFF \
  -DBUILD_TESTING=OFF

cmake --build "$build_dir" --target generate-types
