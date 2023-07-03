package cmd

import (
	"context"
	"github.com/apache/dubbo-admin/pkg/core"
)

type RunCmdOpts struct {
	// The first returned context is closed upon receiving first signal (SIGSTOP, SIGTERM).
	// The second returned context is closed upon receiving second signal.
	// We can start graceful shutdown when first context is closed and forcefully stop when the second one is closed.
	SetupSignalHandler func() (context.Context, context.Context)
}

var DefaultRunCmdOpts = RunCmdOpts{
	SetupSignalHandler: core.SetupSignalHandler,
}
