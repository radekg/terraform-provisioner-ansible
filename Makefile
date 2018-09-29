BINARY_NAME=terraform-provisioner-ansible
PLUGINS_DIR=~/.terraform.d/plugins

.PHONY: plugins-dir
plugins-dir:
	mkdir -p ${PLUGINS_DIR}

.PHONY: lint
lint:
	@which golint > /dev/null || go get -u golang.org/x/lint/golint
	golint

.PHONY: update-dependencies
update-dependencies:
	@which glide > /dev/null || go get -u github.com/Masterminds/glide
	glide up

.PHONY: check-golang-version
check-golang-version:
	./bin/check-golang-version.sh

.PHONY: build-linux
build-linux: check-golang-version plugins-dir
	CGO_ENABLED=0 GOOS=linux installsuffix=cgo go build -o ./${BINARY_NAME}-linux
	cp ./${BINARY_NAME}-linux ${PLUGINS_DIR}/${BINARY_NAME}
	rm ./${BINARY_NAME}-linux

.PHONY: build-darwin
build-darwin: check-golang-version plugins-dir
	CGO_ENABLED=0 GOOS=darwin installsuffix=cgo go build -o ./${BINARY_NAME}-darwin
	cp ./${BINARY_NAME}-darwin ${PLUGINS_DIR}/${BINARY_NAME}
	rm ./${BINARY_NAME}-darwin

# this rule must not be used directly
# this rule is invoked by the bin/build-release-binaries.sh script inside of a docker container where the build happens
.PHONY: build-release
build-release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 installsuffix=cgo go build -o ${GOPATH}/bin/${BINARY_NAME}-linux-amd64_${RELEASE_VERSION}
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 installsuffix=cgo go build -o ${GOPATH}/bin/${BINARY_NAME}-darwin-amd64_${RELEASE_VERSION}

.PHONY: test
test:
	go test

.PHONY: test-verbose
test-verbose:
	go test -v ./...