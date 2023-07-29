FROM golang:1.20-alpine3.18 as builder

ARG GOOS
ARG GOARCH
ENV GOOS=${GOOS}
ENV GOARCH=${GOARCH}

WORKDIR /gatewayd
COPY . /gatewayd

RUN apk --no-cache add git make
RUN mkdir -p dist
RUN make build-${GOOS}-${GOARCH}

FROM alpine:3.17 as runner

ARG GOOS
ARG GOARCH
ENV GOOS=${GOOS}
ENV GOARCH=${GOARCH}

COPY --from=builder /gatewayd/dist/${GOOS}-${GOARCH}/gatewayd /usr/bin/
COPY --from=builder /gatewayd/gatewayd.yaml /etc/gatewayd.yaml
COPY --from=builder /gatewayd/gatewayd_plugins.yaml /etc/gatewayd_plugins.yaml

ENTRYPOINT ["/usr/bin/gatewayd"]
