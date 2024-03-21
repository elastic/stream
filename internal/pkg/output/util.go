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

func Initialize(opts *Options, logger *zap.SugaredLogger, ctx context.Context) (Output, error) {
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
