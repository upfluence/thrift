#!/bin/bash

set -x

tag=$(git describe --tag)

./bootstrap.sh

REPO_VERSION=${tag#v}-upfluence ./configure --enable-ilbs --with-c-glib=off \
                                            --with-cpp=off


make

pushd lib
make generate-types
popd

make clean
