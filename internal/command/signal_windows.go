// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

//go:build windows

package command

import (
	"fmt"
	"os"
	"syscall"
)

// lookupSignal resolves a signal name, such as "SIGINT", to the signal it
// names. Windows only supports the signals that the Go runtime emulates, so the
// set recognized here is much smaller than on other platforms.
func lookupSignal(name string) (os.Signal, error) {
	switch name {
	case "SIGINT":
		return os.Interrupt, nil
	case "SIGTERM":
		return syscall.SIGTERM, nil
	default:
		return nil, fmt.Errorf("signal %v is not supported on windows, use SIGINT or SIGTERM", name)
	}
}
