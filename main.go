package main

import (
	"context"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("merged_camera"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) error {
	mergedCameraModule, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}

	mergedCameraModule.AddModelFromRegistry(ctx, camera.API, model)

	err = mergedCameraModule.Start(ctx)
	defer mergedCameraModule.Close(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
