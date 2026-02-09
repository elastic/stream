FROM golang:1.25.7-alpine3.23 AS builder

RUN apk add --no-cache musl-dev gcc libpcap libpcap-dev

ADD . /app

WORKDIR /app

RUN go mod download

RUN go build

# ------------------------------------------------------------------------------
FROM alpine:3.23

RUN apk add --no-cache libpcap

COPY --chown=0:0 --from=builder /app/stream /stream

ENTRYPOINT ["/stream"]
