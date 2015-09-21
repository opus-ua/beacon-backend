#!/bin/bash

export GOPATH=`pwd`

if [ -v "${VERSION_NUMBER}" ]; then
    VERSION_NUMBER=0.0.0
fi

go install -v -ldflags "-X main.version=$VERSION_NUMBER" github.com/opus-ua/beacon-backend
go test -ldflags "-X main.version=$VERSION_NUMBER" github.com/opus-ua/beacon-backend -v
