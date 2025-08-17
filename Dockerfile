FROM golang:1.25.0-alpine3.22 AS builder

RUN apk add --no-cache musl-dev gcc libpcap libpcap-dev

ADD . /app

WORKDIR /app

RUN go mod download

RUN go build

# ------------------------------------------------------------------------------
FROM alpine:3.22

RUN apk add --no-cache libpcap

COPY --chown=0:0 --from=builder /app/stream /stream

ENTRYPOINT ["/stream"]
