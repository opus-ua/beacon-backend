all: bin/beacon-backend

SRC=$(shell find ./src -name "*.go")

bin/beacon-backend: $(SRC)
	mkdir -p bin
	./build.sh

.PHONY: install
install:
	cp ./bin/beacon-backend /usr/bin/beacon
	cp ./beacon.service /etc/systemd/system/beacon.service
	systemctl daemon-reload
	systemctl enable beacon
	systemctl start beacon

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
