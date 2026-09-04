# build stage
# Built on the runner's own platform and cross-compiled from there: the Go
# toolchain does that far faster than emulating the target platform.
FROM --platform=${BUILDPLATFORM} golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
# GOARM only matters for 32-bit arm (v7: a Pi running a 32-bit OS); every
# other GOARCH ignores it, so it can be set unconditionally.
ARG TARGETOS TARGETARCH TARGETVARIANT
# The release workflow passes the version from the VERSION file; a plain
# docker build stays "dev".
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -ldflags="-w -s -X main.version=${VERSION}" -o /app/eventrouter

# final stage
# A multi-arch manifest, so buildx pulls the variant matching the target
# platform. Nothing is executed in this stage, only copied into it, which is
# why the build needs no QEMU.
FROM quay.io/prometheus/busybox:latest
COPY --from=builder /app/eventrouter /app/eventrouter
COPY docs/config.json /etc/eventrouter/config.json

USER nobody

CMD ["/app/eventrouter"]
