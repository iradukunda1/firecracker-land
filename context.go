package main

import (
	"context"

	lgg "github.com/sirupsen/logrus"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
)

// set logger into context
func ctxSetLogger(ctx context.Context, logger *lgg.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// get logger from context
func ctxGetLogger(ctx context.Context) *lgg.Logger {
	return ctx.Value(loggerKey).(*lgg.Logger)
}
