BINARY_NAME=terraform-provisioner-ansible
PLUGINS_DIR=~/.terraform.d/plugins

.PHONY: plugins-dir
plugins-dir:
	mkdir -p ${PLUGINS_DIR}

.PHONY: install-glide
install-glide:
	@which glide > /dev/null || go get github.com/Masterminds/glide

.PHONY: install
install: install-glide
	glide install

.PHONY: build-linux
build-linux: plugins-dir
	CGO_ENABLED=0 GOOS=linux installsuffix=cgo go build -o ./${BINARY_NAME}-linux
	cp ./${BINARY_NAME}-linux ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: build-darwin
build-darwin: plugins-dir
	CGO_ENABLED=0 GOOS=darwin installsuffix=cgo go build -o ./${BINARY_NAME}-darwin
	cp ./${BINARY_NAME}-darwin ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: test
test:
	go test

.PHONY: test-verbose
test-verbose:
	go test -v