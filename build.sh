#!/bin/bash

OLDGOPATH=$GOPATH
export GOPATH=`pwd`

if [ "$#" == 1 ]; then
    VERSION_NUMBER="$1"
else
    VERSION_NUMBER=0.0.0
fi

HASH=$(git rev-parse HEAD)
LDFLAGS="-X main.version=$VERSION_NUMBER -X main.gitHash=$HASH"

rm -f bin/*
go get gopkg.in/redis.v3
go install -v -ldflags "$LDFLAGS"  github.com/opus-ua/beacon-backend
go test github.com/opus-ua/beacon-backend -v
go test github.com/opus-ua/beacon-post -v

export GOPATH=$OLDGOPATH
