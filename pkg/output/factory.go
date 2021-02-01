// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

var registry = map[string]Factory{}

type Factory func(options *Options) (Output, error)

type Output interface {
	DialContext(ctx context.Context) error
	io.WriteCloser
}

func Register(protocol string, factory Factory) {
	registry[protocol] = factory
}

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

func Available() []string {
	outputs := make([]string, 0, len(registry))
	for k := range registry {
		outputs = append(outputs, k)
	}
	sort.Strings(outputs)
	return outputs
}
