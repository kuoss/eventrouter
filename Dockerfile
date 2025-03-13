FROM golang:1.23 AS base
WORKDIR /temp/
COPY . ./
RUN go mod download -x
RUN CGO_ENABLED=0 go build -ldflags=-w -o /app/eventrouter

FROM debian:bookworm-slim
COPY --from=base /app /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
USER nobody:nobody

CMD ["/bin/sh", "-c", "/app/eventrouter -v 3 -logtostderr"]
