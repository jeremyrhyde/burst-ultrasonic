package main

import (
	"context"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("burst_ultrasonic"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) error {
	burstUltrasonicModule, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}

	burstUltrasonicModule.AddModelFromRegistry(ctx, camera.API, model)

	err = burstUltrasonicModule.Start(ctx)
	defer burstUltrasonicModule.Close(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
