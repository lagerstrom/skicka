ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
UNAME:=$(shell uname)
BIN_DIR = "bin"


.PHONY: dirs
dirs:
	@mkdir -p $(BIN_DIR)

.PHONY: build
build: dirs
	@cd src; \
	$(GOPATH)/bin/packr2; \
	packr2 build -o ../bin/skicka
	@echo "build successful"

.PHONY: clean
clean:
	@if [ -d "bin" ]; then rm -r bin; fi
	@cd src; packr2 clean
	@echo "clean successful"

.PHONY: setup
setup:
	go get -u github.com/gobuffalo/packr/v2/packr2
	@echo ""
	@echo "setup successful"

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
