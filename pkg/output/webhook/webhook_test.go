// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/stream/pkg/output"
)

const (
	username          = "john"
	password          = "password123"
	secretHeaderValue = "foobar"
	contentType       = "application/json"
)

func TestWebhook(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, secretHeaderValue, r.Header.Get("Secret"))

		user, pass, ok := r.BasicAuth()
		assert.True(t, ok, "expected basic auth")
		assert.Equal(t, username, user)
		assert.Equal(t, password, pass)

		if r.Method == http.MethodPost {
			assert.Equal(t, contentType, r.Header.Get("Content-Type"))

			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var event map[string]string
			err = json.Unmarshal(data, &event)
			require.NoError(t, err)
			assert.Equal(t, "hello world!", event["message"])
		}
	}))
	defer ts.Close()

	out, err := New(&output.Options{
		Addr: ts.URL + "/logs",
		WebhookOptions: output.WebhookOptions{
			ContentType: contentType,
			Headers: []string{
				"Secret=" + secretHeaderValue,
			},
			Username: username,
			Password: password,
		},
	})
	require.NoError(t, err)

	err = out.DialContext(context.Background())
	require.NoError(t, err)

	event := map[string]interface{}{
		"message": "hello world!",
	}
	data, err := json.Marshal(event)
	require.NoError(t, err)

	n, err := out.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
}
