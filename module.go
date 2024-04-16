//go:build linux

// package main implements the Revolution Pi module that is supported by Viam
package main

import (
	"context"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
	"viam-labs/viam-revolution-pi/revolutionpi"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("revolution_pi"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) (err error) {
	customModule, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}

	err = customModule.AddModelFromRegistry(ctx, board.API, revolutionpi.Model)
	if err != nil {
		return err
	}

	err = customModule.Start(ctx)
	defer customModule.Close(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
