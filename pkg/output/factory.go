package output

import (
	"context"
	"fmt"
	"io"
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
