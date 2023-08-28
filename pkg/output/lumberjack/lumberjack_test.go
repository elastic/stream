// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lumberjack

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lumber/server"

	"github.com/elastic/stream/pkg/output"
)

func TestSplitAddress(t *testing.T) {
	testCases := []struct {
		address string
		scheme  string
		host    string
		port    string
		fail    bool
	}{
		{
			address: "localhost:5044",
			scheme:  "tcp",
			host:    "localhost",
			port:    "5044",
		},
		{
			address: "127.0.0.1:5044",
			scheme:  "tcp",
			host:    "127.0.0.1",
			port:    "5044",
		},
		{
			address: "[2001:db8:4006:812::200e]:5044",
			scheme:  "tcp",
			host:    "2001:db8:4006:812::200e",
			port:    "5044",
		},
		{
			address: "tcp://localhost:5044",
			scheme:  "tcp",
			host:    "localhost",
			port:    "5044",
		},
		{
			address: "TCP://localhost:5044",
			scheme:  "tcp",
			host:    "localhost",
			port:    "5044",
		},
		{
			address: "tls://localhost:5044",
			scheme:  "tls",
			host:    "localhost",
			port:    "5044",
		},
		{
			address: "localhost",
			fail:    true,
		},
		{
			address: "tcp://localhost",
			fail:    true,
		},
		{
			address: "tls://localhost",
			fail:    true,
		},
		{
			address: "tls:// localhost:5044",
			fail:    true,
		},
		{
			address: "bad://localhost:5044",
			fail:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.address, func(t *testing.T) {
			scheme, host, port, err := splitAddress(tc.address)
			if tc.fail {
				require.Error(t, err)
				t.Log(err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.scheme, scheme)
			assert.Equal(t, tc.host, host)
			assert.Equal(t, tc.port, port)
		})
	}
}

func TestOutputWrite(t *testing.T) {
	// Start server on ephemeral port.
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	s, err := server.NewWithListener(l, server.V2(true))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Receive messages in the background.
	var messages []interface{}
	go func() {
		for batch := range s.ReceiveChan() {
			messages = append(messages, batch.Events...)
			batch.ACK()
		}
	}()

	// Start the lumberjack output that is under test.
	o, err := New(&output.Options{Addr: l.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}

	if err := o.DialContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	const message = "hello world"
	n, err := o.Write([]byte(message))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(message), n)

	// Verify one message received.
	assert.Len(t, messages, 1)
}
