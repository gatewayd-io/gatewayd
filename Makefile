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
	@go mod tidy && CGO_ENABLED=0 go build -tags embed_plugin_template,embed_swagger -trimpath -ldflags "-s -w -X ${CONFIG_PACKAGE}.Version=${VERSION} -X ${CMD_PACKAGE}.UsageReportURL=localhost:59091"

create-build-dir:
	@mkdir -p dist

build-release: tidy create-build-dir
	@echo "Building gatewayd ${VERSION} for release"
	@$(MAKE) build-platform GOOS=linux GOARCH=amd64 OUTPUT_DIR=dist/linux-amd64
	@$(MAKE) build-platform GOOS=linux GOARCH=arm64 OUTPUT_DIR=dist/linux-arm64
	@$(MAKE) build-platform GOOS=darwin GOARCH=amd64 OUTPUT_DIR=dist/darwin-amd64
	@$(MAKE) build-platform GOOS=darwin GOARCH=arm64 OUTPUT_DIR=dist/darwin-arm64
	@$(MAKE) build-platform GOOS=windows GOARCH=amd64 OUTPUT_DIR=dist/windows-amd64
	@$(MAKE) build-platform GOOS=windows GOARCH=arm64 OUTPUT_DIR=dist/windows-arm64

build-platform: tidy
	@echo "Building gatewayd ${VERSION} for $(GOOS)-$(GOARCH)"
	@mkdir -p $(OUTPUT_DIR)
	@cp README.md LICENSE gatewayd.yaml gatewayd_plugins.yaml $(OUTPUT_DIR)/
	@if [ "$(GOOS)" = "windows" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags embed_plugin_template,embed_swagger,windows -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o $(OUTPUT_DIR)/gatewayd.exe; \
		sha256sum $(OUTPUT_DIR)/gatewayd.exe | sed 's#$(OUTPUT_DIR)/##g' >> $(OUTPUT_DIR)/checksum.txt; \
		zip -q -r dist/gatewayd-$(GOOS)-$(GOARCH)-${VERSION}.zip -j $(OUTPUT_DIR)/; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags embed_plugin_template,embed_swagger -trimpath -ldflags "-s -w ${EXTRA_LDFLAGS}" -o $(OUTPUT_DIR)/gatewayd; \
		sha256sum $(OUTPUT_DIR)/gatewayd | sed 's#$(OUTPUT_DIR)/##g' >> $(OUTPUT_DIR)/checksum.txt; \
		tar czf dist/gatewayd-$(GOOS)-$(GOARCH)-${VERSION}.tar.gz -C $(OUTPUT_DIR)/ ${FILES}; \
	fi
	@sha256sum dist/gatewayd-$(GOOS)-$(GOARCH)-${VERSION}.* | sed 's#dist/##g' >> dist/checksums.txt

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
	@go run -tags embed_plugin_template,embed_swagger main.go run --dev

run-race: tidy
	@go run -race -tags embed_plugin_template,embed_swagger main.go run --dev

run-tracing: tidy
	@go run -tags embed_plugin_template,embed_swagger main.go run --tracing --dev

clean:
	@go clean -testcache
	@rm -rf dist

test:
	@go test -tags embed_plugin_template -v -cover -coverprofile=c.out ./...

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

golint:
	@golangci-lint run --show-stats