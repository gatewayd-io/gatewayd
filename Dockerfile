# syntax=docker/dockerfile:1

# Use the official golang image to build the binary.
FROM golang:1.22-alpine3.19 as builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM

WORKDIR /gatewayd
COPY . /gatewayd

RUN apk --no-cache add git make
RUN mkdir -p dist
RUN make build-platform GOOS=${TARGETOS} GOARCH=${TARGETARCH} OUTPUT_DIR=dist/${TARGETOS}-${TARGETARCH}

# Use alpine to create a minimal image to run the gatewayd binary.
FROM alpine:3.19 as runner

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder /gatewayd/dist/${TARGETOS}-${TARGETARCH}/gatewayd /usr/bin/
COPY --from=builder /gatewayd/gatewayd.yaml /etc/gatewayd.yaml
COPY --from=builder /gatewayd/gatewayd_plugins.yaml /etc/gatewayd_plugins.yaml

ENTRYPOINT ["/usr/bin/gatewayd"]
