NAMESPACE=example
NAME=airflow
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

default: build

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	cp ${BINARY} ~/.terraform.d/plugins/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

clean:
	rm -f ${BINARY}

test:
	go test ./... -v

fmt:
	go fmt ./...

vet:
	go vet ./...

.PHONY: build install clean test fmt vet
