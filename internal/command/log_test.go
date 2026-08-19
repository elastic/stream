// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/elastic/stream/internal/output"
)

func newLogTestRunner(t *testing.T, tmpl bool) *logRunner {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return &logRunner{
		logger:   zap.NewNop().Sugar(),
		cmd:      cmd,
		out:      &output.Options{MaxLogLineSize: 500 * 1024},
		template: tmpl,
	}
}

func writeLog(t *testing.T, lines string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(path, []byte(lines), 0o600))
	return path
}

func TestSendLog(t *testing.T) {
	t.Run("sends lines verbatim without --template", func(t *testing.T) {
		// A brace expression must be left untouched when templating is off.
		const line = `{"eventTime":"{{ now }}"}`
		path := writeLog(t, line+"\n")

		r := newLogTestRunner(t, false)
		out := &memoryOutput{}
		require.NoError(t, r.sendLog(path, out))

		require.Equal(t, []string{line}, out.payloads())
	})

	t.Run("renders each line with --template", func(t *testing.T) {
		const line = `{"eventTime":"{{ (now "-720h").Format "2006-01-02T15:04:05Z07:00" }}"}`
		path := writeLog(t, line+"\n")
		want := time.Now().UTC().Add(-720 * time.Hour)

		r := newLogTestRunner(t, true)
		out := &memoryOutput{}
		require.NoError(t, r.sendLog(path, out))

		payloads := out.payloads()
		require.Len(t, payloads, 1)

		var result struct {
			EventTime string `json:"eventTime"`
		}
		require.NoError(t, json.Unmarshal([]byte(payloads[0]), &result))
		ts, err := time.Parse(time.RFC3339, result.EventTime)
		require.NoError(t, err)
		assert.WithinDuration(t, want, ts, time.Minute)
	})

	t.Run("reports an error for an invalid template", func(t *testing.T) {
		path := writeLog(t, `{{ now "bogus" }}`+"\n")

		r := newLogTestRunner(t, true)
		err := r.sendLog(path, &memoryOutput{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "line 1")
	})
}
