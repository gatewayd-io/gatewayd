PROJECT_URL=github.com/gatewayd-io/gatewayd
CONFIG_PACKAGE=${PROJECT_URL}/config
CMD_PACKAGE=${PROJECT_URL}/cmd
LAST_TAGGED_COMMIT=$(shell git rev-list --tags --max-count=1)
LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short ${LAST_TAGGED_COMMIT})
# LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short HEAD)
VERSION=$(shell git describe --tags ${LAST_TAGGED_COMMIT})
TIMESTAMP=$(shell date -u +"%FT%T%z")
VERSION_DETAILS=${TIMESTAMP}/${LAST_TAGGED_COMMIT_SHORT}
EXTRA_LDFLAGS=-X ${CONFIG_PACKAGE}.Version=${VERSION} -X ${CONFIG_PACKAGE}.VersionDetails=${VERSION_DETAILS} -X ${CMD_PACKAGE}.UsageReportURL=usage.gatewayd.io:59091
FILES=gatewayd README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml

tidy:
	@go mod tidy

build-dev:
	@go mod tidy && CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w -X ${CONFIG_PACKAGE}.Version=${VERSION} -X ${CMD_PACKAGE}.UsageReportURL=localhost:59091"

build-release: tidy
	@mkdir -p dist

	@echo "Building gatewayd ${VERSION} for linux-amd64"
	@mkdir -p dist/linux-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-amd64/
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-amd64/gatewayd
	@tar czf dist/gatewayd-linux-amd64-${VERSION}.tar.gz -C ./dist/linux-amd64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for linux-arm64"
	@mkdir -p dist/linux-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-arm64/
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-arm64/gatewayd
	@tar czf dist/gatewayd-linux-arm64-${VERSION}.tar.gz -C ./dist/linux-arm64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for darwin-amd64"
	@mkdir -p dist/darwin-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-amd64/
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-amd64/gatewayd
	@tar czf dist/gatewayd-darwin-amd64-${VERSION}.tar.gz -C ./dist/darwin-amd64/ ${FILES}

	@echo "Building gatewayd ${VERSION} for darwin-arm64"
	@mkdir -p dist/darwin-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-arm64/
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-arm64/gatewayd
	@tar czf dist/gatewayd-darwin-arm64-${VERSION}.tar.gz -C ./dist/darwin-arm64/ ${FILES}

	@echo "Generating checksums"
	@sha256sum dist/gatewayd-linux-amd64-${VERSION}.tar.gz | sed 's/dist\///g' > dist/checksums.txt
	@sha256sum dist/gatewayd-linux-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-darwin-amd64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-darwin-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

build-linux-packages:
	@echo "Building gatewayd ${VERSION} for linux-amd64"
	@VERSION=${VERSION} GOARCH=amd64 envsubst < nfpm.yaml > nfpm-current.yaml
	@nfpm package -t dist -p deb -f nfpm-current.yaml

	@echo "Building gatewayd ${VERSION} for linux-arm64"
	@VERSION=${VERSION} GOARCH=arm64 envsubst < nfpm.yaml > nfpm-current.yaml
	@nfpm package -t dist -p deb -f nfpm-current.yaml

	@echo "Building gatewayd ${VERSION} for linux-amd64"
	@VERSION=${VERSION} GOARCH=amd64 envsubst < nfpm.yaml > nfpm-current.yaml
	@nfpm package -t dist -p rpm -f nfpm-current.yaml

	@echo "Building gatewayd ${VERSION} for linux-arm64"
	@VERSION=${VERSION} GOARCH=arm64 envsubst < nfpm.yaml > nfpm-current.yaml
	@nfpm package -t dist -p rpm -f nfpm-current.yaml

	@rm nfpm-current.yaml

	@echo "Generating checksums"
	@sha256sum dist/gatewayd_${VERSION}_amd64.deb | sed 's/dist\///g' > dist/checksums.txt
	@sha256sum dist/gatewayd_${VERSION}_arm64.deb | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-${VERSION}.x86_64.rpm | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-${VERSION}.aarch64.rpm | sed 's/dist\///g' >> dist/checksums.txt

run: tidy
	@go run -tags embed_swagger main.go run --dev

run-race: tidy
	@go run -race -tags embed_swagger main.go run --dev

run-tracing: tidy
	@go run -tags embed_swagger main.go run --tracing --dev

clean:
	@go clean -testcache
	@rm -rf dist

test:
	@go test -v ./...

update-all:
	@go get -u ./...
	@go mod tidy

lint:
	@buf lint

gen:
	@buf generate

update:
	@buf mod update
