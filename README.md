# stream

[![Build Status](https://beats-ci.elastic.co/job/Library/job/stream-mbp/job/main/badge/icon)](https://beats-ci.elastic.co/job/Library/job/stream-mbp/job/main/)

stream is a test utility for streaming data via:

- UDP
- TCP
- TLS
- Webhook
- GCP Pub-Sub
- Kafka
- HTTP Mock Server

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

- `rules`: a list of rules. More restrictive rules need to go on top.
- `path`: the path to match. It can use [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux#pkg-overview) parameters patterns.
- `methods`: a list of methods to match with the rule.
- `user` and `password`: username and password for basic auth matching.
- `query_params`: Key-Value definitions of the query parameters to match. It can use [gorilla/mux](https://pkg.go.dev/github.com/gorilla/mux#Route.Queries) parameters patterns for the values. Web form params will also be added and compared against this for simplicity.
- `request_headers`: Key-Value definitions of the headers to match. Any headers outside of this list will be ignored. The matches can be defined [as regular expressions](https://pkg.go.dev/github.com/gorilla/mux#Route.HeadersRegexp).
- `request_body`: a string defining the expected body to match for the request. If the string is quoted with slashes, the leading and trailing slash are stripped and the resulting string is interpreted as a regular expression.
- `responses`: a list of zero or more responses to return on matches. If more than one are set, they will be returned in rolling sequence.
- `status_code`: the status code to return.
- `headers`: Key-Value list of the headers to return with the response. The values will be evaluated as [Go templates](https://golang.org/pkg/text/template/).
- `body`: a string defining the body that will be returned as a response. It will be evaluated as a [Go template](https://golang.org/pkg/text/template/).

When using [Go templates](https://golang.org/pkg/text/template/) as part of the `response.headers` or `response.body`, some functions and data will be available:

- `hostname`: function that returns the hostname.
- `env KEY`: function that returns the KEY from environment.
- `sum A B`: function that returns the sum of numbers A and B (only for integers).
- `file PATH`: function that returns the contents of the file at PATH.
- `.req_num`: variable containing the current request number, auto incremented after every request for the rule.
- `.request.vars`: map containing the variables received in the request (both query and form).
- `.request.url`: the url object. Can be used as per [the Go URL documentation.](https://golang.org/pkg/net/url/#URL)
- `.request.headers` the headers object. Can be used as per [the Go http.Header documentation.](https://golang.org/pkg/net/http/#Header)
