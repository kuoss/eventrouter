FROM golang:1.23 AS base
WORKDIR /temp/
COPY . ./
RUN go mod download -x
RUN CGO_ENABLED=0 go build -ldflags=-w -o /eventrouter

FROM gcr.io/distroless/static-debian12:latest
COPY --from=base /eventrouter /eventrouter

RUN mkdir -p /etc/eventrouter && \
    echo '{"sink": "stdout"}' > /etc/eventrouter/config.json && \
    chmod 644 /etc/eventrouter/config.json

USER nobody

CMD ["/eventrouter", "-v", "3", "-logtostderr"]
