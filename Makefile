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

.PHONY: clean
clean:
	rm -rf bin
