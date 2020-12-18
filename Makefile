ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR = "bin"

.PHONY: dirs
dirs:
	@mkdir -p $(BIN_DIR)

.PHONY: build
build: dirs
	@cd src; \
	packr2; \
	go build -o ../bin/skicka
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
