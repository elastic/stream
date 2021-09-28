FROM golang:1.16.8 as builder

RUN apt-get update \
       && apt-get install -y libpcap-dev \
       && rm -rf /var/lib/apt/lists/*

ADD . /app

WORKDIR /app

RUN go mod download

RUN go build

# ------------------------------------------------------------------------------
FROM debian:stable-slim

COPY --chown=0:0 --from=builder /usr/lib/x86_64-linux-gnu/libpcap.so.0.8 /usr/lib/x86_64-linux-gnu/libpcap.so.0.8
COPY --chown=0:0 --from=builder /usr/lib/x86_64-linux-gnu/libpcap.so.1.10.0 /usr/lib/x86_64-linux-gnu/libpcap.so.1.10.0
COPY --chown=0:0 --from=builder /app/stream /stream

ENTRYPOINT ["/stream"]
