package output

import "time"

type Options struct {
	Addr     string
	Protocol string
	Delay    time.Duration
	Retries  int
}
