BINARY_NAME=terraform-provisioner-ansible
PLUGINS_DIR=~/.terraform.d/plugins

.PHONY: plugins-dir
plugins-dir:
	mkdir -p ${PLUGINS_DIR}

.PHONY: install-dep
install-dep:
	@which dep > /dev/null || go get -u github.com/golang/dep/cmd/dep

.PHONY: install
install: install-dep
	dep ensure

.PHONY: update-dependencies
update-dependencies:
	dep ensure -update

.PHONY: build-linux
build-linux: plugins-dir
	CGO_ENABLED=0 GOOS=linux installsuffix=cgo go build -o ./${BINARY_NAME}-linux
	cp ./${BINARY_NAME}-linux ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: build-darwin
build-darwin: plugins-dir
	CGO_ENABLED=0 GOOS=darwin installsuffix=cgo go build -o ./${BINARY_NAME}-darwin
	cp ./${BINARY_NAME}-darwin ${PLUGINS_DIR}/${BINARY_NAME}

.PHONY: build-windows
build-darwin: plugins-dir
	CGO_ENABLED=0 GOOS=windows installsuffix=cgo go build -o ./${BINARY_NAME}-windows.exe
	cp ./${BINARY_NAME}-windows ${PLUGINS_DIR}/${BINARY_NAME}.exe

.PHONY: test
test:
	go test

.PHONY: test-verbose
test-verbose:
	go test -v