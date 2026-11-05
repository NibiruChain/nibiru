#!/bin/sh
#
# Invoke Mockery v2 to update generated mocks for the given type.
#
# v2.49.2 (the upstream pin) no longer builds with Go 1.27: its
# golang.org/x/tools v0.17.0 dependency fails to compile. v2.53.8 generates
# byte-identical mocks for every interface in this tree.

go run github.com/vektra/mockery/v2@v2.53.8 --disable-version-string --case underscore --name "$*"
