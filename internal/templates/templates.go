// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package templates holds the Go text/template helper functions shared by
// stream's templated inputs (e.g. the log command's --template mode) and the
// http-server mock config. Keeping them in one place ensures a template
// behaves identically no matter where it is evaluated.
package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Funcs returns the template function map available to stream templates.
//
// The following functions and their behaviour are documented in the README:
//
//   - env KEY:      returns the KEY environment variable.
//   - hostname:     returns the current hostname.
//   - sum A B:      returns the sum of two integers.
//   - file PATH:    returns the contents of the file at PATH.
//   - glob PATTERN: returns the names of files matching PATTERN.
//   - minify_json:  compacts a JSON document onto a single line.
//   - now [OFFSET]: returns the current UTC time, optionally offset by a Go
//     duration string (e.g. "-720h" for 30 days ago).
func Funcs() template.FuncMap {
	return template.FuncMap{
		"env":         env,
		"hostname":    hostname,
		"sum":         sum,
		"file":        file,
		"glob":        filepath.Glob,
		"minify_json": minify,
		"now":         now,
	}
}

// Render parses s as a Go text/template using Funcs and executes it with no
// data, returning the rendered bytes. Missing keys render as their zero value,
// matching the behaviour of the http-server config templates.
func Render(s string) ([]byte, error) {
	t, err := template.New("").
		Option("missingkey=zero").
		Funcs(Funcs()).
		Parse(s)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func env(key string) string {
	return os.Getenv(key)
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func sum(a, b int) int {
	return a + b
}

func file(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func minify(body string) (string, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(json.RawMessage(body))
	return strings.TrimSpace(buf.String()), err
}

// now returns the current UTC time. An optional Go duration string
// offsets the result (e.g. "-720h" for 30 days ago). The returned
// time.Time value exposes its methods to templates, so callers can
// format it as needed: {{ (now).Format "2006-01-02" }}.
func now(offset ...string) (time.Time, error) {
	t := time.Now().UTC()
	if len(offset) == 0 {
		return t, nil
	}
	d, err := time.ParseDuration(offset[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %w", offset[0], err)
	}
	return t.Add(d), nil
}
