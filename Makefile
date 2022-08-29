ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
UNAME:=$(shell uname)
BIN_DIR = "bin"


.PHONY: dirs
dirs:
	@mkdir -p $(BIN_DIR)

.PHONY: build
build: dirs
	go build -o $(BIN_DIR)/skicka cmd/skicka.go
	@echo "build successful"

.PHONY: clean
clean:
	@if [ -d "bin" ]; then rm -r bin; fi
	@echo "clean successful"

.PHONY: build-docker
build-docker:
	docker build -t skicka .

.PHONY: run-docker
run-docker:
	@echo -n "Your local IP is: "
ifeq ($(UNAME),Linux)
	@ip route get 1.1.1.1 | grep -oP 'src \K\S+'
endif
	@docker run -it -v /tmp/skicka:/tmp/skicka -p 8000:8000 skicka
