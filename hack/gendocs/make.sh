#!/usr/bin/env bash

pushd $GOPATH/src/stash.appscode.dev/cli/hack/gendocs
go run main.go
popd
