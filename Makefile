PROJECT_URL=github.com/gatewayd-io/gatewayd
CONFIG_PACKAGE=${PROJECT_URL}/config
CMD_PACKAGE=${PROJECT_URL}/cmd
LAST_TAGGED_COMMIT=$(shell git rev-list --tags --max-count=1)
LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short ${LAST_TAGGED_COMMIT})
# LAST_TAGGED_COMMIT_SHORT=$(shell git rev-parse --short HEAD)
VERSION=$(shell git describe --tags ${LAST_TAGGED_COMMIT})
TIMESTAMP=$(shell date -u +"%FT%T%z")
VERSION_DETAILS=${TIMESTAMP}/${LAST_TAGGED_COMMIT_SHORT}
EXTRA_LDFLAGS=-X ${CONFIG_PACKAGE}.Version=${VERSION} -X ${CONFIG_PACKAGE}.VersionDetails=${VERSION_DETAILS} -X ${CMD_PACKAGE}.UsageReportURL=usage.gatewayd.io:443
FILES=gatewayd README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml

tidy:
	@go mod tidy

build-dev:
	@go mod tidy && CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w -X ${CONFIG_PACKAGE}.Version=${VERSION} -X ${CMD_PACKAGE}.UsageReportURL=localhost:59091"

create-build-dir:
	@mkdir -p dist

build-windows-amd64: tidy
	@echo "Building gatewayd ${VERSION} for windows-amd64"
	@mkdir -p dist/windows-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/windows-amd64/
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_swagger,windows -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/windows-amd64/gatewayd.exe
	@zip -r dist/gatewayd-windows-amd64-${VERSION}.zip -j ./dist/windows-amd64/
	@sha256sum dist/gatewayd-windows-amd64-${VERSION}.zip | sed 's/dist\///g' >> dist/checksums.txt

build-windows-arm64: tidy
	@echo "Building gatewayd ${VERSION} for windows-arm64"
	@mkdir -p dist/windows-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/windows-arm64/
	@GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -tags embed_swagger,windows -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/windows-arm64/gatewayd.exe
	@zip -r dist/gatewayd-windows-arm64-${VERSION}.zip -j ./dist/windows-arm64/
	@sha256sum dist/gatewayd-windows-arm64-${VERSION}.zip | sed 's/dist\///g' >> dist/checksums.txt

build-linux-amd64: tidy
	@echo "Building gatewayd ${VERSION} for linux-amd64"
	@mkdir -p dist/linux-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-amd64/
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-amd64/gatewayd
	@tar czf dist/gatewayd-linux-amd64-${VERSION}.tar.gz -C ./dist/linux-amd64/ ${FILES}
	@sha256sum dist/gatewayd-linux-amd64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

build-linux-arm64: tidy
	@echo "Building gatewayd ${VERSION} for linux-arm64"
	@mkdir -p dist/linux-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/linux-arm64/
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/linux-arm64/gatewayd
	@tar czf dist/gatewayd-linux-arm64-${VERSION}.tar.gz -C ./dist/linux-arm64/ ${FILES}
	@sha256sum dist/gatewayd-linux-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

build-darwin-amd64: tidy
	@echo "Building gatewayd ${VERSION} for darwin-amd64"
	@mkdir -p dist/darwin-amd64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-amd64/
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-amd64/gatewayd
	@tar czf dist/gatewayd-darwin-amd64-${VERSION}.tar.gz -C ./dist/darwin-amd64/ ${FILES}
	@sha256sum dist/gatewayd-darwin-amd64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

build-darwin-arm64: tidy
	@echo "Building gatewayd ${VERSION} for darwin-arm64"
	@mkdir -p dist/darwin-arm64
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml dist/darwin-arm64/
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o dist/darwin-arm64/gatewayd
	@tar czf dist/gatewayd-darwin-arm64-${VERSION}.tar.gz -C ./dist/darwin-arm64/ ${FILES}
	@sha256sum dist/gatewayd-darwin-arm64-${VERSION}.tar.gz | sed 's/dist\///g' >> dist/checksums.txt

build-release: create-build-dir build-windows-amd64 build-windows-arm64 build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

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
	@sha256sum dist/gatewayd_$(VERSION:v%=%)_amd64.deb | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd_$(VERSION:v%=%)_arm64.deb | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-$(VERSION:v%=%).x86_64.rpm | sed 's/dist\///g' >> dist/checksums.txt
	@sha256sum dist/gatewayd-$(VERSION:v%=%).aarch64.rpm | sed 's/dist\///g' >> dist/checksums.txt

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
	@go test -v -cover -coverprofile=c.out ./...

test-race:
	@go test -race -v ./...

benchmark:
	@go test -bench=. -benchmem -run=^# ./...

update-all:
	@go get -u ./...
	@go mod tidy

lint:
	@buf lint

gen:
	@buf generate

update:
	@buf mod update

gen-docs:
	@go run main.go gen-docs