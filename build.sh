#!/bin/bash

export GOPATH=`pwd`

if [ "$#" == 1 ]; then
    VERSION_NUMBER="$1"
else
    VERSION_NUMBER=0.0.0
fi

HASH=$(git rev-parse HEAD)
LDFLAGS="-X main.version=$VERSION_NUMBER -X main.gitHash=$HASH"
go install -v -ldflags "$LDFLAGS"  github.com/opus-ua/beacon-backend
go test github.com/opus-ua/beacon-backend -v
