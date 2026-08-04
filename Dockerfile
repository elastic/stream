FROM golang:1.26.5-alpine3.23 AS builder

ADD . /app

WORKDIR /app

RUN go mod download

# stream is pure Go, so build a static binary that depends on neither libc nor
# libpcap and can run in an image with no userland at all.
RUN CGO_ENABLED=0 go build

# ------------------------------------------------------------------------------
FROM scratch

# The static binary needs no operating system, but the TLS based outputs need
# root certificates in order to verify the certificates servers present.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --chown=0:0 --from=builder /app/stream /stream

ENTRYPOINT ["/stream"]
