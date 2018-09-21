BINARY_NAME=terraform-provisioner-ansible
PLUGINS_DIR=~/.terraform.d/plugins

.PHONY: plugins-dir
plugins-dir:
	mkdir -p ${PLUGINS_DIR}

.PHONY: tools
tools:
	@which golint > /dev/null || go get -u golang.org/x/lint/golint
	@which glide > /dev/null || go get -u github.com/Masterminds/glide

.PHONY: lint
lint:
	golint

.PHONY: install
install: tools
	glide install

.PHONY: update-dependencies
update-dependencies:
	glide up

.PHONY: check-golang-version
check-golang-version:
	./bin/check-golang-version.sh

.PHONY: build-linux
build-linux: check-golang-version plugins-dir
	CGO_ENABLED=0 GOOS=linux installsuffix=cgo go build -o ./${BINARY_NAME}-linux
	cp ./${BINARY_NAME}-linux ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: build-darwin
build-darwin: check-golang-version plugins-dir
	CGO_ENABLED=0 GOOS=darwin installsuffix=cgo go build -o ./${BINARY_NAME}-darwin
	cp ./${BINARY_NAME}-darwin ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: test
test:
	go test

.PHONY: test-verbose
test-verbose:
	go test -v