// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package httpserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/stream/pkg/log"
	"github.com/elastic/stream/pkg/output"
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

	server, err := New(&opts, logger.Sugar())
	require.NoError(t, err)

	t.Cleanup(func() { server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, server.Start(ctx))

	addr := server.listener.Addr().(*net.TCPAddr).String()

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
}
