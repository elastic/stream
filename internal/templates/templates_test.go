// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package templates

import (
	"encoding/json"
	"testing"
	"time"
)

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

func TestRender(t *testing.T) {
	t.Run("no template", func(t *testing.T) {
		const in = `{"message":"hello"}`
		got, err := Render(in)
		if err != nil {
			t.Fatalf("Render(%q) error: %v", in, err)
		}
		if string(got) != in {
			t.Errorf("Render(%q) = %q; want unchanged", in, got)
		}
	})

	t.Run("now with offset formats", func(t *testing.T) {
		const in = `{"eventTime":"{{ (now "-720h").Format "2006-01-02T15:04:05Z07:00" }}"}`
		want := time.Now().UTC().Add(-720 * time.Hour)

		got, err := Render(in)
		if err != nil {
			t.Fatalf("Render(%q) error: %v", in, err)
		}

		var result struct {
			EventTime string `json:"eventTime"`
		}
		if err := json.Unmarshal(got, &result); err != nil {
			t.Fatalf("unmarshal(%s) error: %v", got, err)
		}
		ts, err := time.Parse(time.RFC3339, result.EventTime)
		if err != nil {
			t.Fatalf("time.Parse(%q) error: %v", result.EventTime, err)
		}
		if ts.Before(want.Add(-time.Minute)) || ts.After(want.Add(time.Minute)) {
			t.Errorf("eventTime = %s; want within 1m of %s", ts, want)
		}
	})

	t.Run("invalid template", func(t *testing.T) {
		if _, err := Render(`{{ now "bogus" }}`); err == nil {
			t.Error("Render with invalid offset error = nil; want error")
		}
	})
}
