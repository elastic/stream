// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package log provides a utility for creating a new zap.Logger.
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates and returns a new zap.Logger configured for production use
// with ISO8601 time encoding and DebugLevel logging. It returns an error if the
// logger cannot be built.
func NewLogger() (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	conf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	log, err := conf.Build()
	if err != nil {
		return nil, err
	}
	return log, nil
}
