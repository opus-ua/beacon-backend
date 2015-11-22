all: bin/beacon test

SRC=$(shell find ./src -name "*.go")

VERSION_NUMBER=0.0.0
GOPATH:=$(GOPATH):`pwd`
HASH:=$(shell git rev-parse HEAD)
DEBUG_GOOGLE_ID:=$(shell cat debug.id)
RELEASE_GOOGLE_ID:=$(shell cat release.id)
LDFLAGS=-X main.version=$(VERSION_NUMBER) -X main.gitHash=$(HASH) -X main.releaseGoogleID=$(RELEASE_GOOGLE_ID) -X main.debugGoogleID=$(DEBUG_GOOGLE_ID)


bin/beacon: $(SRC)
	mkdir -p bin
	GOPATH=$(GOPATH) go get gopkg.in/redis.v3
	GOPATH=$(GOPATH) go install -v -ldflags "$(LDFLAGS)"  github.com/opus-ua/beacon

.PHONY: test
test:
	GOPATH=$(GOPATH) go test github.com/opus-ua/beacon -v --bench .
	GOPATH=$(GOPATH) go test github.com/opus-ua/beacon-db -v --bench .

.PHONY: install
install:
	cp ./bin/beacon /usr/bin/beacon
	cp ./tools/beacon.service /etc/systemd/system/beacon.service
	systemctl daemon-reload
	systemctl enable beacon
	systemctl start beacon

.PHONY: format
format:
	GOPATH=$(GOPATH) go fmt src/github.com/opus-ua/beacon/*
	GOPATH=$(GOPATH) go fmt src/github.com/opus-ua/beacon-post/*
	GOPATH=$(GOPATH) go fmt src/github.com/opus-ua/beacon-db/*

.PHONY: package
package:
	mkdir -p ./beacon
	cp ./bin/beacon ./beacon/beacon
	cp ./tools/beacon.service ./beacon/beacon.service
	cp ./tools/install.sh beacon/install.sh
	tar czf ./bin/beacon.tar.gz ./beacon
	rm -rf ./beacon

.PHONY: clean
clean:
	rm -rf bin
