# build stage
FROM --platform=${BUILDPLATFORM} golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
ARG TARGETPLATFORM
RUN CGO_ENABLED=0 GOARCH="${TARGETPLATFORM#*/}" go build -ldflags="-w -s" -o /app/eventrouter

# final stage
FROM quay.io/prometheus/busybox-linux-${TARGETARCH}:latest
COPY --from=builder /app/eventrouter /app/eventrouter
COPY docs/config.json /etc/eventrouter/config.json

USER nobody

CMD ["/app/eventrouter", "-v", "3", "-logtostderr"]
