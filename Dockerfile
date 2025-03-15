FROM golang:1.23 AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s -trimpath" -o /eventrouter
RUN echo '{"sink": "stdout"}' > /config.json

FROM gcr.io/distroless/static-debian12:latest
COPY --from=base /eventrouter /eventrouter
COPY --from=base /config.json /etc/eventrouter/config.json

USER nobody
CMD ["/eventrouter", "-v", "3", "-logtostderr"]
