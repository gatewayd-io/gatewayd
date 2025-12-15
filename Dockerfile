# syntax=docker/dockerfile:1

# Use the official golang image to build the binary.
FROM golang:1.25-alpine3.21 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM

WORKDIR /gatewayd
COPY . /gatewayd

RUN apk --no-cache add \
    'git~=2.47' \
    'make~=4.4' \
    'openssl~=3.3' && \
    mkdir -p dist && \
    make build-platform GOOS=${TARGETOS} GOARCH=${TARGETARCH} OUTPUT_DIR=dist/${TARGETOS}-${TARGETARCH}

# Use alpine to create a minimal image to run the gatewayd binary.
FROM alpine:3.21 AS runner

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder /gatewayd/dist/${TARGETOS}-${TARGETARCH}/gatewayd /usr/bin/
COPY --from=builder /gatewayd/gatewayd.yaml /etc/gatewayd.yaml
COPY --from=builder /gatewayd/gatewayd_plugins.yaml /etc/gatewayd_plugins.yaml

ENTRYPOINT ["/usr/bin/gatewayd"]
