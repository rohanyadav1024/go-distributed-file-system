package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
)

func main() {
	cfg := config.Load()

	if err := logging.Init(cfg.Log); err != nil {
		panic(err)
	}

	ctx := context.Background() //initalize background context
	ctx = logging.WithRequestID(ctx, ids.NewRequestID()) //Manipulating context: add request ID to context for logging

	logging.FromContext(ctx).Info("meta starting up")

	//make a buffered channel to listen for OS signals
	sigCh := make(chan os.Signal, 1)

	// Notify the channel on interrupt and terminate signals
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	// SIGINT -> Interrupt signal (Ctrl+C)
	// SIGTERM -> Termination signal (graceful shutdown) DOCKER STOP

	// Block until a signal is received
	<-sigCh

	//flush any buffered log entries
	_ = logging.L().Sync() 
}
