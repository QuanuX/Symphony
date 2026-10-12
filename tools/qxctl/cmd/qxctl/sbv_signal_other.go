//go:build !darwin && !linux

package main

import (
	"context"
	"os"
	"os/signal"
)

func sbvSignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}
