# stream

[![Build Status](https://beats-ci.elastic.co/job/Library/job/stream-mbp/job/main/badge/icon)](https://beats-ci.elastic.co/job/Library/job/stream-mbp/job/main/)

stream is a test utility for streaming data via:

- UDP
- TCP
- TLS
- Webhook
- GCP Pub-Sub
- Kafka
- [Lumberjack](#lumberjack-output-reference)
- HTTP Mock Server
- Azure Blob Storage
- Google Cloud Storage
- Azure Event Hub

Input data can be read from:

- log file - Newline delimited files are streamed line by line.
- pcap file - Each packet's transport layer payload is streamed as a packet.
  Useful for replaying netflow and IPFIX captures.

## HTTP Server mock reference

`stream` can also serve logs setting up a complete HTTP mock server.

Usage:

```bash
stream http-server --addr=":8080" --config="./config.yml"
```

The server can be configured to serve specific log files
on certain routes. The config should be defined in a yaml file of the following format:

```yaml
---
    rules:
    - path: "/path1/test"
      methods: ["GET"]

      user: username
      password: passwd
      query_params:
        p1: ["v1"]
      request_headers:
        accept: ["application/json"]

      responses:
      - headers:
          x-foo: ["test"]
        status_code: 200
        body: |-
          {"next": "http://{{ hostname }}/page/{{ sum (.req_num) 1 }}"}
    - path: "/page/{pagenum:[0-9]}" params.
      methods: ["POST"]

      responses:
      - status_code: 200
        body: "{{ .request.vars.pagenum }}"
        headers:
          content-type: ["text/plain"]
```

The rules will be defined in order, and will only match if all criteria is true for a request. This means that you need to define the more restrictive rules on top.

### Options

- `as_sequence`: if this is set to `true`, the server will exit with an error if the requests are not performed in order.
- `rules`: a list of rules. More restrictive rules need to go on top.
- `path`: the path to match. It can use [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux#pkg-overview) parameters patterns.
- `methods`: a list of methods to match with the rule.
- `user` and `password`: username and password for basic auth matching.
- `query_params`: Key-Value definitions of the query parameters to match. It can use [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux#Route.Queries) parameters patterns for the values. Web form params will also be added and compared against this for simplicity. If a key is given an empty value, requests with this parameter will not satisfy the rule.
- `request_headers`: Key-Value definitions of the headers to match. Any headers outside of this list will be ignored. The matches can be defined [as regular expressions](https://pkg.go.dev/github.com/gorilla/mux#Route.HeadersRegexp).
- `request_body`: a string defining the expected body to match for the request. If the string is quoted with slashes, the leading and trailing slash are stripped and the resulting string is interpreted as a regular expression.
- `responses`: a list of zero or more responses to return on matches. If more than one are set, they will be returned in rolling sequence. If `as_sequence` is set to `true`, they will only be able to be hit once instead of in rolling sequence.
- `status_code`: the status code to return.
- `headers`: Key-Value list of the headers to return with the response. The values will be evaluated as [Go templates](https://golang.org/pkg/text/template/).
- `body`: a string defining the body that will be returned as a response. It will be evaluated as a [Go template](https://golang.org/pkg/text/template/).

When using [Go templates](https://golang.org/pkg/text/template/) as part of the `response.headers` or `response.body`, some functions and data will be available:

- `hostname`: function that returns the hostname.
- `env KEY`: function that returns the KEY from environment.
- `sum A B`: function that returns the sum of numbers A and B (only for integers).
- `file PATH`: function that returns the contents of the file at PATH.
- `glob PATTERN`: function that returns the names of all files matching glob PATTERN (see [filepath.Match](https://pkg.go.dev/path/filepath#Match) for syntax).
- `.req_num`: variable containing the current request number, auto incremented after every request for the rule.
- `.request.vars`: map containing the variables received in the request (both query and form).
- `.request.url`: the url object. Can be used as per [the Go URL documentation.](https://golang.org/pkg/net/url/#URL)
- `.request.headers` the headers object. Can be used as per [the Go http.Header documentation.](https://golang.org/pkg/net/http/#Header)

### Fault injection

The `http-server` subcommand supports fault injection, a form of chaos
engineering, to test the resilience of client applications. It allows you to
simulate failures and delays in the server's responses. This is useful for
identifying how your client application behaves under adverse network conditions
or when a service it depends on is experiencing problems.

There are two types of fault injection available, and they can be used
independently or together:

1. **HTTP Error Injection**: This injects an HTTP error status code in a certain
   fraction of responses.
2. **Response Delay**: This adds a delay to a certain fraction of responses.

These fault injection settings apply to all mocked API paths that the
`http-server` is listening on.

#### HTTP Error Injection

You can configure the server to return a specific HTTP error code for a portion
of the requests it receives.

* `--fault-rate`: A floating-point number between `0.0` and `1.0` that specifies
  the fraction of requests that should fail. For example, a value of `0.1`
  means 10% of requests will receive an error. The default is `0.0` (no errors
  injected).
* `--fault-error-code`: The HTTP status code to return for the failed requests.
  The default is `500` (Internal Server Error).

Example: To make 25% of requests fail with a `503 Service Unavailable` error:

```bash
stream http-server --fault-rate 0.25 --fault-error-code 503
```

#### Response Delay

You can introduce a delay in the server's response for a portion of the
requests.

* `--delay-rate`: A floating-point number between `0.0` and `1.0` that specifies
  the fraction of requests that should be delayed. The default is `0.0` (no
  delay).
* `--delay-duration`: The duration of the delay to apply to requests (e.g.,
  `500ms`, `2s`). The default is `0s`.

Example: To add a 2-second delay to 50% of requests:

```bash
stream http-server --delay-rate 0.5 --delay-duration 2s
```

## Lumberjack Output Reference

Lumberjack is the protocol used between Elastic Beats and Logstash. It is
implemented using the [elastic/go-lumber](https://github.com/elastic/go-lumber)
library. `stream` sends data using version 2 of the Lumberjack protocol. Each
log line is sent as its own batch containing a single event. The output blocks
until the batch is ACKed.

When using the Lumberjack output the address flag value (`--addr`) can indicate
when to send via TLS. Format the address as a URL with a `tls` scheme
(e.g. `tls://127.0.0.1:5044`) to use TLS. If a scheme isn't specified then a
TCP connection is used (i.e. `localhost:5044` implies `tcp://localhost:5044`).

By default, Lumberjack batches contain one event with a `message` field.

```json
[
  {
    "message": "{{ input_data }}"
  }
]
```

If `--lumberjack-parse-json` is used then the input data is parsed as JSON
and the resulting data is sent as a batch.

## GCS Output Reference

The GCS output is used to collect data from the configured source, create a GCS bucket, and populate it with the incoming data.
When specifying a (`--addr`) which should be a combination of both host and port, usually pointing to a locally running emulator,
the client will be overriding the configured API endpoint, which defaults to the public google storage API, towards the emulator instead.
The emulator does not require authentication.

### Options

- `gcs-bucket`: The name of the GCS bucket that should be created, should not already exist.
- `gcs-object`: The name of the GCS object that will be populated with the collected data, using the configured GCS bucket.
- `gcs-projectid`: The related projectID used when creating the bucket, this is required to be changed from the default value when not using an emulator.

## Azure Event Hub Output Reference

The Azure Event Hub output is used to collect data from the azure event hub resource
When specifying a `--azure-event-hub-connection-string`, it should be retrieved as described [here](https://learn.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string).

Sample config:

```yml
version: '2.3'
services:
  azure-event-hub:
    image: docker.elastic.co/observability/stream:v0.12.0
    volumes:
      - ./sample_logs:/sample_logs:ro
    command:
      - log
      - --retry=30
      - -p=azureeventhub
      - --azure-event-hub-connection-string="Endpoint=sb://test-eventhub-stream-seis.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SharedAccessKey"
      - /sample_logs/testdata.log
```

### Options

- `azure-event-hub-connection-string`: The connection string to connect to the Event Hub.
- `azure-event-hub-namespace`: The fully qualified domain name of the Event Hubs namespace. This it the Event Hubs namespace followed by `servicebus.windows.net` (e.g. myeventhub.servicebus.windows.net).
- `azure-event-hub-name`: The name of the Event hub.
