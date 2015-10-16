all: bin/beacon-backend

SRC=$(shell find ./src -name "*.go")

VERSION_NUMBER=0.0.0
GOPATH:=$(GOPATH):`pwd`
HASH:=$(shell git rev-parse HEAD)
LDFLAGS=-X main.version=$(VERSION_NUMBER) -X main.gitHash=$(HASH)

bin/beacon-backend: $(SRC) test
	mkdir -p bin
	GOPATH=$(GOPATH) go get gopkg.in/redis.v3
	GOPATH=$(GOPATH) go install -v -ldflags "$(LDFLAGS)"  github.com/opus-ua/beacon-backend

.PHONY: test
test:
	GOPATH=$(GOPATH) go test github.com/opus-ua/beacon-backend -v
	GOPATH=$(GOPATH) go test github.com/opus-ua/beacon-post -v

.PHONY: install
install:
	cp ./bin/beacon-backend /usr/bin/beacon
	cp ./tools/beacon.service /etc/systemd/system/beacon.service
	systemctl daemon-reload
	systemctl enable beacon
	systemctl start beacon

.PHONY: format
format:
	GOPATH=$(GOPATH) go fmt src/github.com/opus-ua/beacon-backend/*

.PHONY: package
package:
	mkdir -p ./beacon
	cp ./bin/beacon-backend ./beacon/beacon
	cp ./tools/beacon.service ./beacon/beacon.service
	cp ./tools/install.sh beacon/install.sh
	tar czf ./bin/beacon.tar.gz ./beacon
	rm -rf ./beacon

.PHONY: clean
clean:
	rm -rf bin
