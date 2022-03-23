FROM golang:1.17.8 as builder

RUN apt-get update \
       && apt-get install -y libpcap-dev \
       && rm -rf /var/lib/apt/lists/*

ADD . /app

WORKDIR /app

RUN go mod download

RUN go build

# ------------------------------------------------------------------------------
FROM debian:stable-slim

RUN apt-get update \
       && apt-get install -y libpcap0.8 \
       && rm -rf /var/lib/apt/lists/*

COPY --chown=0:0 --from=builder /app/stream /stream

ENTRYPOINT ["/stream"]
