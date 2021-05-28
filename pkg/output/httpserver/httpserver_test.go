// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package httpserver

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/stream/pkg/output"
)

func TestHTTPServer(t *testing.T) {
	cases := []struct {
		description     string
		opts            output.HTTPServerOptions
		input           []string
		expectedOutput  []string
		expectedHeaders http.Header
	}{
		{
			description:    "can get one log per response",
			input:          []string{"a", "b", "c"},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			description: "returns expected response headers",
			opts: output.HTTPServerOptions{
				ResponseHeaders: []string{"content-type", "custom"},
			},
			input:          []string{"a"},
			expectedOutput: []string{"a"},
			expectedHeaders: http.Header{
				"Content-Type": []string{"custom"},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			out, err := New(&output.Options{
				Addr:              "127.0.0.1:1111",
				HTTPServerOptions: tc.opts,
			})

			require.NoError(t, err)
			require.NoError(t, out.DialContext(context.Background()))

			for i, in := range tc.input {
				var n int
				var werr error
				var wg sync.WaitGroup
				trigger := make(chan struct{})
				wg.Add(1)
				go func(in string) {
					defer wg.Done()

					timeout := time.NewTimer(time.Second)
					defer timeout.Stop()

					select {
					case <-timeout.C:
					default:
						close(trigger)
						n, werr = out.Write([]byte(in))
					}
				}(in)

				<-trigger
				
				resp, err := http.Get("http://127.0.0.1:1111")
				require.NoError(t, err)
				t.Cleanup(func() { resp.Body.Close() })

				wg.Wait()
				require.NoError(t, werr)
				assert.Equal(t, len(in), n)

				body, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)

				assert.Equal(t, tc.expectedOutput[i], string(body))

				for h, vs := range tc.expectedHeaders {
					assert.EqualValues(t, vs, resp.Header[h])
				}
			}

			require.NoError(t, out.Close())
		})
	}
}
