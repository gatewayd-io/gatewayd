PACKAGE_NAME=github.com/gatewayd-io/gatewayd/config
LAST_TAGGED_COMMIT=$(shell git rev-list --tags --max-count=1)
LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short ${LAST_TAGGED_COMMIT})
# LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short HEAD)
VERSION=$(shell git describe --tags ${LAST_TAGGED_COMMIT})
TIMESTAMP=$(shell date -u +"%FT%T%z")
VERSION_DETAILS=${TIMESTAMP}/${LAST_TAGGED_COMMIT_SHORT}
EXTRA_LDFLAGS=-X ${PACKAGE_NAME}.Version=${VERSION} -X ${PACKAGE_NAME}.VersionDetails=${VERSION_DETAILS}
FILES=gatewayd README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml

tidy:
	@go mod tidy

build-dev:
	@go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X ${PACKAGE_NAME}.Version=${VERSION}"

build-release: tidy
	@mkdir -p dist

	@echo "Building gatewayd ${VERSION} for linux-amd64"
	@mkdir -p dist/linux-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-amd64/
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-amd64/gatewayd
	@tar czf dist/gatewayd-linux-amd64-${VERSION}.tar.gz -C ./dist/linux-amd64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for linux-arm64"
	@mkdir -p dist/linux-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-arm64/
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-arm64/gatewayd
	@tar czf dist/gatewayd-linux-arm64-${VERSION}.tar.gz -C ./dist/linux-arm64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for darwin-amd64"
	@mkdir -p dist/darwin-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-amd64/
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-amd64/gatewayd
	@tar czf dist/gatewayd-darwin-amd64-${VERSION}.tar.gz -C ./dist/darwin-amd64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for darwin-arm64"
	@mkdir -p dist/darwin-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-arm64/
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-arm64/gatewayd
	@tar czf dist/gatewayd-darwin-arm64-${VERSION}.tar.gz -C ./dist/darwin-arm64/ ${FILES}

	@echo "Generating checksums"
	@sha256sum dist/gatewayd-linux-amd64-${VERSION}.tar.gz | sed 's/dist\///g' > dist/checksums.txt
	@sha256sum dist/gatewayd-linux-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-darwin-amd64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-darwin-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

run: tidy
	@go run main.go run

clean:
	@go clean -testcache
	@rm -rf dist

test:
	@go test -v ./...

update-all:
	@go get -u ./...
