// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package output

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/elastic/go-concert/timed"
)

// Initialize creates and configures a new Output using the provided Options and
// logger, then attempts to establish a connection with retries. It returns the
// connected Output or an error if the connection could not be established within
// the allowed retries or if the provided context is canceled. The logger is used
// for informational and debug messages during initialization and connection
// attempts.
func Initialize(ctx context.Context, opts *Options, logger *zap.SugaredLogger) (Output, error) {
	o, err := New(opts)
	if err != nil {
		return nil, err
	}

	var dialErr error
	for i := 0; i < opts.Retries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		logger.Debug("Connecting...")
		if dialErr = o.DialContext(ctx); dialErr != nil {
			if err = timed.Wait(ctx, time.Second); err != nil {
				return nil, err
			}
			continue
		}

		break
	}
	if dialErr != nil {
		return nil, dialErr
	}
	logger.Info("Connected")

	return o, nil
}
