FROM golang:1.23-alpine AS base
WORKDIR /temp/
COPY . ./
RUN go mod download -x
RUN CGO_ENABLED=0 go build -ldflags=-w -o /app/eventrouter

FROM alpine:3.20
COPY --from=base /app /app
RUN apk update --no-cache && apk add ca-certificates
WORKDIR /app
USER nobody:nobody

CMD ["/bin/sh", "-c", "/app/eventrouter -v 3 -logtostderr"]
