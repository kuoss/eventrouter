# build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /eventrouter

# final stage
FROM gcr.io/distroless/static-debian12:latest
COPY --from=builder /eventrouter /eventrouter
COPY docs/config.json /etc/eventrouter/config.json

USER nobody

CMD ["/eventrouter", "-v", "3", "-logtostderr"]
