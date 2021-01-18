package output

import (
	"context"
	"time"

	"github.com/elastic/go-concert/timed"
	"go.uber.org/zap"
)

func Initialize(opts *Options, logger *zap.SugaredLogger, ctx context.Context) (Output, error) {
	o, err := New(opts)
	if err != nil {
		return nil, err
	}

	if opts.Delay > 0 {
		logger.Debugw("Delaying connection.", "delay", opts.Delay)
		if err = timed.Wait(ctx, opts.Delay); err != nil {
			return nil, err
		}
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
