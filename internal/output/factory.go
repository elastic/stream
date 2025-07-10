// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package output provides an interface for writing data to various destinations
// (outputs) in stream. An output is responsible for delivering data, such as log
// lines or events, to external systems including files, cloud storage, message
// queues, or network endpoints. Each output implementation encapsulates
// connection, delivery, and error handling logic specific to its protocol or
// service.
package output

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

var registry = map[string]Factory{}

// Factory is a function that creates a new Output.
type Factory func(options *Options) (Output, error)

// Output is an io.WriteCloser that can be dialed.
type Output interface {
	// DialContext dials the output.
	DialContext(ctx context.Context) error
	io.WriteCloser
}

// Register registers a new output factory for a given protocol.
// This is not thread-safe and should only be called from init() functions.
func Register(protocol string, factory Factory) {
	registry[protocol] = factory
}

// New creates a new Output for the given options.
// It looks up the protocol in the registry and calls the corresponding factory.
func New(opts *Options) (Output, error) {
	if opts.Protocol == "" {
		return nil, fmt.Errorf("protocol is required")
	}

	factory, found := registry[strings.ToLower(opts.Protocol)]
	if !found {
		return nil, fmt.Errorf("unknown protocol %q", opts.Protocol)
	}

	return factory(opts)
}

// Available returns a sorted list of registered output protocols.
func Available() []string {
	outputs := make([]string, 0, len(registry))
	for k := range registry {
		outputs = append(outputs, k)
	}
	sort.Strings(outputs)
	return outputs
}
