#!/bin/bash

export GOPATH=`pwd`

if [ "$#" == 1 ]; then
    VERSION_NUMBER="$1"
else
    VERSION_NUMBER=0.0.0
fi

echo $VERSION_NUMBER

go install -v -ldflags "-X main.version=$VERSION_NUMBER" github.com/opus-ua/beacon-backend
go test github.com/opus-ua/beacon-backend -v
