// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package httpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/elastic/stream/pkg/output"
)

type Server struct {
	logger   *zap.SugaredLogger
	opts     *Options
	listener net.Listener
	server   *http.Server
	ctx      context.Context
}

type Options struct {
	*output.Options
	TLSCertificate string        // TLS certificate file path.
	TLSKey         string        // TLS key file path.
	ReadTimeout    time.Duration // HTTP Server read timeout.
	WriteTimeout   time.Duration // HTTP Server write timeout.
	ConfigPath     string        // Config path.
}

func New(opts *Options, logger *zap.SugaredLogger) (*Server, error) {
	if opts.Addr == "" {
		return nil, errors.New("a listen address is required")
	}

	if !(opts.TLSCertificate == "" && opts.TLSKey == "") &&
		!(opts.TLSCertificate != "" && opts.TLSKey != "") {
		return nil, errors.New("both TLS certificate and key files must be defined")
	}

	config, err := newConfigFromFile(opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	handler, err := newHandlerFromConfig(config, logger)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
		Handler:        handler,
	}

	return &Server{
		logger: logger,
		opts:   opts,
		server: server,
	}, nil
}

func (o *Server) Start(ctx context.Context) error {
	o.ctx = ctx

	l, err := net.Listen("tcp", o.opts.Addr)
	if err != nil {
		if l, err = net.Listen("tcp6", o.opts.Addr); err != nil {
			return fmt.Errorf("failed to listen on address: %v", err)
		}
	}

	o.listener = l

	if o.opts.TLSCertificate != "" && o.opts.TLSKey != "" {
		go func() { o.logger.Info(o.server.ServeTLS(l, o.opts.TLSCertificate, o.opts.TLSKey).Error()) }()
	} else {
		go func() { o.logger.Info(o.server.Serve(l).Error()) }()
	}

	o.logger.Debugf("listening on %s", o.listener.Addr().(*net.TCPAddr).String())

	return nil
}

func (o *Server) Close() error {
	o.logger.Info("shutting down http-server...")

	ctx, cancel := context.WithTimeout(o.ctx, time.Second)
	defer cancel()

	return o.server.Shutdown(ctx)
}

func newHandlerFromConfig(config *config, logger *zap.SugaredLogger) (http.Handler, error) {
	router := mux.NewRouter()

	var buf bytes.Buffer

	for i, rule := range config.Rules {
		rule := rule
		var count int
		i := i
		logger.Debugf("Setting up rule #%d for path %q", i, rule.Path)
		route := router.HandleFunc(rule.Path, func(w http.ResponseWriter, r *http.Request) {
			response := func() *response {
				switch len(rule.Responses) {
				case 0:
					return nil
				case 1:
					return &rule.Responses[0]
				}
				return &rule.Responses[count%len(rule.Responses)]
			}()

			count += 1

			logger.Debug(fmt.Sprintf("Rule #%d matched: request #%d => %s", i, count, strRequest(r)))

			data := map[string]interface{}{
				"req_num": count,
				"request": map[string]interface{}{
					"vars":    mux.Vars(r),
					"url":     r.URL,
					"headers": r.Header,
				},
			}

			if response != nil {
				for k, tpls := range response.Headers {
					for _, tpl := range tpls {
						buf.Reset()
						if err := tpl.Execute(&buf, data); err != nil {
							logger.Errorf("executing header template %s: %s, %v", k, tpl.Root.String(), err)
							continue
						}
						w.Header().Add(k, buf.String())
					}
				}

				w.WriteHeader(response.StatusCode)

				if err := response.Body.Execute(w, data); err != nil {
					logger.Errorf("executing body template %s: %v", response.Body.Root.String(), err)
				}
			}
		})

		route.Methods(rule.Methods...)

		for key, vals := range rule.QueryParams {
			for _, v := range vals {
				route.Queries(key, v)
			}
		}

		for key, vals := range rule.RequestHeaders {
			for _, v := range vals {
				route.HeadersRegexp(key, v)
			}
		}

		route.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
			user, password, _ := r.BasicAuth()
			return rule.User == user && rule.Password == password
		})

		route.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return false
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			return rule.RequestBody == string(body)
		})
	}

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("request did not match with any rule: %s", strRequest(r))
		w.WriteHeader(404)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// merge together form params into the url ones to make checks easier
		_ = r.ParseForm()
		r.URL.RawQuery = r.Form.Encode()

		router.ServeHTTP(w, r)
	}), nil
}

func strRequest(r *http.Request) string {
	var b strings.Builder
	b.WriteString("Request path: ")
	b.WriteString(r.Method)
	b.WriteString(" ")
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
