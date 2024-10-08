#!/bin/bash

set -x

git_tag=$(git describe --tag)
tag=${TAG:-$git_tag}

./bootstrap.sh

REPO_VERSION=${tag#v}-upfluence ./configure --enable-ilbs --with-c-glib=off \
                                            --with-cpp=off


make

pushd lib
make generate-types
popd

make clean
