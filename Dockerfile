# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS builder

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

ENV PORT=8080 \
         GIN_MODE=production

RUN apk add --no-cache git

COPY . /src

WORKDIR /src

RUN case "$TARGETVARIANT" in \
      v6) export GOARM=6 ;; \
      v7) export GOARM=7 ;; \
    esac && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags "-w -s -X main.version=$VERSION" -o /torch ./cmd

FROM alpine:3.23

WORKDIR /bin/

COPY --from=builder /torch ./torch

USER 65532:65532

ENTRYPOINT ["/bin/torch"]
