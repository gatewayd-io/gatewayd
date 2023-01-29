PACKAGE_NAME=github.com/gatewayd-io/gatewayd/config
LAST_TAGGED_COMMIT=$(shell git rev-list --tags --max-count=1)
# LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short ${LAST_TAGGED_COMMIT})
LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short HEAD)
VERSION=$(shell git describe --tags ${LAST_TAGGED_COMMIT})
TIMESTAMP=$(shell date -u +"%FT%T%z")
VERSION_DETAILS=${TIMESTAMP}/${LAST_TAGGED_COMMIT_SHORT}-(dev)
EXTRA_LDFLAGS=-X ${PACKAGE_NAME}.Version=${VERSION}# -X ${PACKAGE_NAME}.VersionDetails=${VERSION_DETAILS}

build:
	@go mod tidy && go build -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}"

run:
	@go mod tidy && go run main.go run

clean:
	@go clean -testcache

test:
	@go test -v ./...
