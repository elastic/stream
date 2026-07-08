// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/stream/internal/log"
	"github.com/elastic/stream/internal/output"
)

func TestHTTPServer(t *testing.T) {
	cfg := `---
    rules:
    - path: "/path1/test"
      methods: ["GET"]

      user: username
      password: passwd
      query_params:
        p1: ["v1"]
        p2: null
      request_headers:
        accept: ["application/json"]

      responses:
      - headers:
          x-foo: ["test"]
        status_code: 200
        body: |-
          {"next": "http://{{ hostname }}/page/{{ sum (.req_num) 1 }}"}

    - path: "/page/{pagenum:[0-9]}"
      methods: ["POST"]

      responses:
      - status_code: 200
        body: "{{ .request.vars.pagenum }}"
        headers:
          content-type: ["text/plain"]

    - path: "/static/minify"
      methods: ["GET"]

      responses:
      - status_code: 200
        body: |-
          {{ minify_json ` + "`" + `
          {
          	"key1": "value1",
          	"key2": "<value2>"
          }
          ` + "`" + `}}

    - path: "/time/now"
      methods: ["GET"]

      responses:
      - status_code: 200
        body: |-
          {"ts": "{{ (now).Format "2006-01-02T15:04:05Z07:00" }}"}

    - path: "/time/offset"
      methods: ["GET"]

      responses:
      - status_code: 200
        body: |-
          {"ts": "{{ (now "-720h").Format "2006-01-02T15:04:05Z07:00" }}"}

    - path: "/orgs/test/audit-log"
      methods: ["GET"]

      responses:
      - status_code: 200
        headers:
          Link:
            - '<http://{{ .request.host }}/orgs/test/audit-log?after=abcd>; rel="next"'
        body: |-
          []
`

	f, err := ioutil.TempFile("", "test")
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(f.Name()) })

	_, err = f.WriteString(cfg)
	require.NoError(t, err)

	opts := Options{
		Options: &output.Options{
			Addr: "localhost:0",
		},
		ConfigPath: f.Name(),
	}

	logger, err := log.NewLogger()
	require.NoError(t, err)

	_, addr := startTestServer(t, &opts, logger.Sugar())

	t.Run("request does not match path unless all requirements are met", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://"+addr+"/path1/test?p1=v1", nil)
		require.NoError(t, err)
		req.Header.Add("accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 404, resp.StatusCode) // must fail because user/pass is missing

		req.SetBasicAuth("username", "passwd")

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode) // should work when all criteria matches

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		hostname, err := os.Hostname()
		require.NoError(t, err)

		assert.JSONEq(t, fmt.Sprintf(`{"next": "http://%s/page/2"}`, hostname), string(body))
		assert.Equal(t, "test", resp.Header.Get("x-foo"))
	})

	t.Run("can map request info in response templates", func(t *testing.T) {
		req, err := http.NewRequest("POST", "http://"+addr+"/page/2", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		assert.Equal(t, 200, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.JSONEq(t, "2", string(body))
		assert.Equal(t, "text/plain", resp.Header.Get("content-type"))
	})

	t.Run("request has rejected parameter", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://"+addr+"/path1/test?p1=v1&p2=bad", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, 404, resp.StatusCode) // must fail because p2 is present

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, []byte{}, body)
	})

	t.Run("minify static JSON", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://"+addr+"/static/minify", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, `{"key1":"value1","key2":"<value2>"}`, string(body))
	})

	t.Run("now returns current time", func(t *testing.T) {
		before := time.Now().UTC()

		req, err := http.NewRequest("GET", "http://"+addr+"/time/now", nil)
		if err != nil {
			t.Fatalf("NewRequest error: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Do error: %v", err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}
		resp.Body.Close()

		// Extract the timestamp from {"ts": "2026-06-17T05:30:00Z"}.
		var result struct{ Ts string }
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", body, err)
		}
		got, err := time.Parse(time.RFC3339, result.Ts)
		if err != nil {
			t.Fatalf("time.Parse(%q) error: %v", result.Ts, err)
		}
		if got.Before(before.Add(-time.Second)) || got.After(time.Now().UTC().Add(time.Second)) {
			t.Errorf("now() = %s; want between %s and now", got, before)
		}
	})

	t.Run("request host is available in response templates", func(t *testing.T) {
		host := "svc-github" + addr[strings.LastIndex(addr, ":"):]
		req, err := http.NewRequest("GET", "http://"+addr+"/orgs/test/audit-log", nil)
		if err != nil {
			t.Fatalf("NewRequest error: %v", err)
		}
		req.Host = host

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Do error: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Fatalf("StatusCode = %d; want 200", resp.StatusCode)
		}

		wantLink := fmt.Sprintf(`<http://%s/orgs/test/audit-log?after=abcd>; rel="next"`, host)
		if got := resp.Header.Get("Link"); got != wantLink {
			t.Fatalf("Link header = %q; want %q", got, wantLink)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}
		resp.Body.Close()

		if got := string(body); got != "[]" {
			t.Fatalf("body = %q; want %q", got, "[]")
		}
	})

	t.Run("now with offset", func(t *testing.T) {
		before := time.Now().UTC().Add(-720 * time.Hour)

		req, err := http.NewRequest("GET", "http://"+addr+"/time/offset", nil)
		if err != nil {
			t.Fatalf("NewRequest error: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Do error: %v", err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}
		resp.Body.Close()

		var result struct{ Ts string }
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", body, err)
		}
		got, err := time.Parse(time.RFC3339, result.Ts)
		if err != nil {
			t.Fatalf("time.Parse(%q) error: %v", result.Ts, err)
		}
		if got.Before(before.Add(-time.Second)) || got.After(before.Add(time.Second)) {
			t.Errorf("now(\"-720h\") = %s; want within 1s of %s", got, before)
		}
	})
}

func TestNow(t *testing.T) {
	t.Run("no offset", func(t *testing.T) {
		before := time.Now().UTC()
		got, err := now()
		if err != nil {
			t.Fatalf("now() error: %v", err)
		}
		if got.Before(before.Add(-time.Second)) || got.After(time.Now().UTC().Add(time.Second)) {
			t.Errorf("now() = %s; want within 1s of current time", got)
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		before := time.Now().UTC().Add(-24 * time.Hour)
		got, err := now("-24h")
		if err != nil {
			t.Fatalf("now(%q) error: %v", "-24h", err)
		}
		if got.Before(before.Add(-time.Second)) || got.After(before.Add(time.Second)) {
			t.Errorf("now(%q) = %s; want within 1s of %s", "-24h", got, before)
		}
	})

	t.Run("positive offset", func(t *testing.T) {
		expected := time.Now().UTC().Add(2 * time.Hour)
		got, err := now("2h")
		if err != nil {
			t.Fatalf("now(%q) error: %v", "2h", err)
		}
		if got.Before(expected.Add(-time.Second)) || got.After(expected.Add(time.Second)) {
			t.Errorf("now(%q) = %s; want within 1s of %s", "2h", got, expected)
		}
	})

	t.Run("invalid offset", func(t *testing.T) {
		_, err := now("bogus")
		if err == nil {
			t.Error("now(\"bogus\") error = nil; want error")
		}
	})
}

func TestRunAsSequence(t *testing.T) {
	cfg := `---
  as_sequence: true
  rules:
  - path: "/path/1"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req1": "{{ .req_num }}"}
    - status_code: 200
      body: |-
        {"req2": "{{ .req_num }}"}
  - path: "/path/2"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req3": "{{ .req_num }}"}
  - path: "/path/3"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req4": "{{ .req_num }}"}
    - status_code: 200
      body: |-
        {"req5": "{{ .req_num }}"}
  - path: "/path/4"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req6": "{{ .req_num }}"}
  - path: "/path/5"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req7": "{{ .req_num }}"}
`

	f, err := ioutil.TempFile("", "test")
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(f.Name()) })

	_, err = f.WriteString(cfg)
	require.NoError(t, err)

	opts := Options{
		Options: &output.Options{
			Addr: "localhost:0",
		},
		ConfigPath: f.Name(),
	}

	logger, err := log.NewLogger()
	logger = logger.WithOptions(zap.OnFatal(zapcore.WriteThenPanic))
	require.NoError(t, err)

	t.Run("requests succeed if made in the expected order", func(t *testing.T) {
		server, addr := startTestServer(t, &opts, logger.Sugar())

		reqTests := []struct {
			path         string
			expectedBody string
		}{
			{"http://" + addr + "/path/1", `{"req1": "1"}`},
			{"http://" + addr + "/path/1", `{"req2": "2"}`},
			{"http://" + addr + "/path/2", `{"req3": "1"}`},
			{"http://" + addr + "/path/3", `{"req4": "1"}`},
			{"http://" + addr + "/path/3", `{"req5": "2"}`},
			{"http://" + addr + "/path/4", `{"req6": "1"}`},
			{"http://" + addr + "/path/5", `{"req7": "1"}`},
		}

		buf := new(bytes.Buffer)
		server.server.Handler = http.HandlerFunc(inspectPanic(server.server.Handler, buf))

		for _, reqTest := range reqTests {
			req, err := http.NewRequest("GET", reqTest.path, nil)
			require.NoError(t, err)
			req.Header.Add("accept", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			resp.Body.Close()

			assert.JSONEq(t, reqTest.expectedBody, string(body))
			assert.Equal(t, "", buf.String())
		}
	})

	t.Run("requests fail if made in the wrong order", func(t *testing.T) {
		server, addr := startTestServer(t, &opts, logger.Sugar())
		buf := new(bytes.Buffer)
		server.server.Handler = http.HandlerFunc(inspectPanic(server.server.Handler, buf))

		req, err := http.NewRequest("GET", "http://"+addr+"/path/1", nil)
		require.NoError(t, err)
		req.Header.Add("accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.JSONEq(t, `{"req1": "1"}`, string(body))

		req, err = http.NewRequest("GET", "http://"+addr+"/path/2", nil)
		require.NoError(t, err)
		req.Header.Add("accept", "application/json")

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		assert.Equal(t, "expecting to match request #1 in sequence, matched rule #2 instead, exiting", buf.String())
	})
}

func TestExitOnUnmatchedRule(t *testing.T) {
	cfg := `---
  rules:
  - path: "/path/1"
    methods: ["GET"]
    request_headers:
      accept: ["application/json"]
    responses:
    - status_code: 200
      body: |-
        {"req1": "{{ .req_num }}"}
`

	f, err := ioutil.TempFile("", "test")
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(f.Name()) })

	_, err = f.WriteString(cfg)
	require.NoError(t, err)

	opts := Options{
		Options: &output.Options{
			Addr: "localhost:0",
		},
		ConfigPath:          f.Name(),
		ExitOnUnmatchedRule: true,
	}

	logger, err := log.NewLogger()
	logger = logger.WithOptions(zap.OnFatal(zapcore.WriteThenPanic))
	require.NoError(t, err)

	server, addr := startTestServer(t, &opts, logger.Sugar())
	buf := new(bytes.Buffer)
	server.server.Handler = http.HandlerFunc(inspectPanic(server.server.Handler, buf))

	req, err := http.NewRequest("GET", "http://"+addr+"/path/2", nil)
	require.NoError(t, err)
	req.Header.Add("accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	assert.Equal(t, "--exit-on-unmatched-rule is set, exiting", buf.String())
}

func inspectPanic(h http.Handler, writer io.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(writer, "%v", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	}
}

func startTestServer(t *testing.T, opts *Options, logger *zap.SugaredLogger) (*Server, string) {
	server, err := New(opts, logger)
	require.NoError(t, err)

	t.Cleanup(func() { server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, server.Start(ctx))

	addr := server.listener.Addr().(*net.TCPAddr).String()

	return server, addr
}
