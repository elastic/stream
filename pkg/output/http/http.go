// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package http

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/elastic/stream/pkg/log"
	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("httpsrv", New)
}

type Output struct {
	logger  *zap.SugaredLogger
	opts    *output.Options
	server  *http.Server
	logChan chan []byte
}

func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("a listen address is required")
	}

	if !(opts.HTTPSrvOptions.TLSCertificate == "" && opts.HTTPSrvOptions.TLSKey == "") &&
		!(opts.HTTPSrvOptions.TLSCertificate != "" && opts.HTTPSrvOptions.TLSKey != "") {
		return nil, errors.New("both TLS certificate and key files must be defined")
	}

	if len(opts.HTTPSrvOptions.ResponseHeaders)%2 != 0 {
		return nil, errors.New("response headers must be a list of pairs")
	}

	logger, err := log.NewLogger()
	if err != nil {
		return nil, err
	}
	slogger := logger.Sugar().With("output", "httpsrv")

	logChan := make(chan []byte)
	server := &http.Server{
		Addr:           opts.Addr,
		ReadTimeout:    opts.HTTPSrvOptions.ReadTimeout,
		WriteTimeout:   opts.HTTPSrvOptions.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
		Handler:        newHandler(opts, logChan, slogger),
	}

	return &Output{
		logger:  slogger,
		opts:    opts,
		server:  server,
		logChan: logChan,
	}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	if o.opts.TLSCertificate != "" && o.opts.TLSKey != "" {
		go func() { o.logger.Info(o.server.ListenAndServeTLS(o.opts.TLSCertificate, o.opts.TLSKey)) }()
	} else {
		go func() { o.logger.Info(o.server.ListenAndServe()) }()
	}
	return nil
}

func (o *Output) Close() error {
	defer close(o.logChan)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = o.server.Shutdown(ctx)
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	select {
	case <-timer.C:
		return 0, errors.New("waiting to write for too long")
	case o.logChan <- b:
		return len(b), nil
	}
}

func newHandler(opts *output.Options, logChan <-chan []byte, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := <-logChan

		defer r.Body.Close()
		logger.Debug(strRequest(r))

		for i := 0; i < len(opts.HTTPSrvOptions.ResponseHeaders); i += 2 {
			w.Header().Add(opts.HTTPSrvOptions.ResponseHeaders[i], opts.HTTPSrvOptions.ResponseHeaders[i+1])
		}

		_, _ = w.Write(b)
	}
}

func strRequest(r *http.Request) string {
	var b strings.Builder
	b.WriteString("Request path: ")
	b.WriteString(r.URL.String())
	b.WriteString(", Request Headers: ")
	for k, v := range r.Header {
		b.WriteString(fmt.Sprintf("'%s: %s' ", k, v))
	}
	b.WriteString(", Request Body: ")
	body, _ := ioutil.ReadAll(r.Body)
	b.Write(body)
	return b.String()
}
