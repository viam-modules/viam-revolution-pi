package main

import (
	"context"
	"time"

	"viam-labs/viam-revolution-pi/revolution_pi"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("revolution_pi"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) (err error) {
	logger.Info("yo mod start")
	custom_module, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}
	logger.Info("yo mod register")
	time.Sleep(5 * time.Second)
	err = custom_module.AddModelFromRegistry(ctx, board.API, revolution_pi.Model)
	if err != nil {
		return err
	}
	logger.Info("yo mod actual start")

	err = custom_module.Start(ctx)
	defer custom_module.Close(ctx)
	if err != nil {
		return err
	}
	logger.Info("yo mod Done")

	<-ctx.Done()
	logger.Info("yo mod Done Done")
	return nil
}
