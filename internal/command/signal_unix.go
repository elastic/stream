// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

//go:build !windows

package command

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// lookupSignal resolves a signal name, such as "SIGUSR1", to the signal it
// names.
func lookupSignal(name string) (os.Signal, error) {
	num := unix.SignalNum(name)
	if num == 0 {
		return nil, fmt.Errorf("unknown signal %v", name)
	}
	return num, nil
}
