// Package main merges the the result of NextPointCloud from multiple cameras
package main

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/rdk/robot/framesystem"
	"go.viam.com/rdk/spatialmath"
)

const (
	timeFormat            = time.RFC3339
	grpcConnectionTimeout = 10 * time.Second
	downloadTimeout       = 30 * time.Second
	maxCacheSize          = 100
)

var (
	// model is the model of a replay camera.
	model = resource.DefaultModelFamily.WithModel("merged_pcd")
)

func init() {
	resource.RegisterComponent(camera.API, model, resource.Registration[camera.Camera, *Config]{
		Constructor: newMergedPCDCamera,
	})
}

// Validate checks that the config attributes are valid for a replay camera.
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Cameras == nil {
		return nil, resource.NewConfigValidationFieldRequiredError(path, "camera")
	}
	deps := cfg.Cameras

	deps = append(deps, framesystem.InternalServiceName.String())

	return deps, nil
}

// Config describes how to configure the merged camera component.
type Config struct {
	Cameras []string `json:"cameras,omitempty"`
}

type mergedPCDCamera struct {
	resource.Named
	logger logging.Logger

	cameras []camera.Camera
	mu      sync.Mutex

	fsService framesystem.Service

	closed bool
}

// newPCDCamera creates a new replay camera based on the inputted config and dependencies.
func newMergedPCDCamera(
	ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger,
) (camera.Camera, error) {
	cam := &mergedPCDCamera{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := cam.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return cam, nil
}

// Close stops replay camera, closes the channels and its connections to the cloud.
func (merged *mergedPCDCamera) Close(ctx context.Context) error {
	merged.mu.Lock()
	defer merged.mu.Unlock()

	merged.closed = true
	return nil
}

// Reconfigure finishes the bring up of the replay camera by evaluating given arguments and setting up the required cloud
// connection.
func (merged *mergedPCDCamera) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {

	mergedPCDCameraConfig, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	var cameras []camera.Camera
	for _, cameraName := range mergedPCDCameraConfig.Cameras {

		cam, err := camera.FromDependencies(deps, cameraName)
		if err != nil {
			return errors.Wrapf(err, "error getting camera %v", cameraName)
		}

		// If there is a camera provided in the 'camera' field, we enforce that it supports PCD.
		properties, err := cam.Properties(ctx)
		if err != nil {
			return errors.Wrapf(err, "error getting camera properties %v", cameraName)
		}

		if properties.SupportsPCD != true {
			return errors.Errorf("error camera %v does not support PCDs", cameraName)
		}

		cameras = append(cameras, cam)
	}

	for name, dep := range deps {
		if name == framesystem.InternalServiceName {
			fsService, ok := dep.(framesystem.Service)
			if !ok {
				return errors.New("frame system service is invalid type")
			}
			merged.fsService = fsService
			break
		}
	}

	merged.cameras = cameras
	return nil
}

// NextPointCloud returns the next point cloud retrieved from cloud storage based on the applied filter.
func (merged *mergedPCDCamera) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	merged.mu.Lock()
	defer merged.mu.Unlock()
	if merged.closed {
		return nil, errors.New("session closed")
	}

	var cloudAndOffsetFuncs []pointcloud.CloudAndOffsetFunc
	for _, cam := range merged.cameras {
		cloudAndOffsetFunc := func(ctx context.Context) (pointcloud.PointCloud, spatialmath.Pose, error) {
			pc, err := cam.NextPointCloud(ctx)

			// determine transform from each camera to first camera
			origin := referenceframe.NewPoseInFrame(merged.cameras[0].Name().ShortName(), spatialmath.NewZeroPose())
			transformedPose, err := merged.fsService.TransformPose(ctx, origin, cam.Name().ShortName(), nil)
			if err != nil {
				return nil, nil, errors.Errorf("issue getting tranform from camera %v to first camera %v", merged.cameras[0].Name().ShortName(), cam.Name().ShortName())
			}

			return pc, transformedPose.Pose(), err
		}

		cloudAndOffsetFuncs = append(cloudAndOffsetFuncs, cloudAndOffsetFunc)
	}

	mergedPC, err := pointcloud.MergePointClouds(ctx, cloudAndOffsetFuncs, merged.logger)
	if err != nil {
		return nil, errors.Wrapf(err, "issue merging pointclouds")
	}

	return mergedPC, err

}

// Images is a part of the camera interface but is not implemented for replay.
func (merged *mergedPCDCamera) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	return nil, resource.ResponseMetadata{}, errors.New("Images is unimplemented")
}

// Properties is a part of the camera interface and returns the camera.Properties struct with SupportsPCD set to true.
func (merged *mergedPCDCamera) Properties(ctx context.Context) (camera.Properties, error) {
	props := camera.Properties{
		SupportsPCD: true,
	}
	return props, nil
}

// Projector is a part of the camera interface but is not implemented for replay.
func (merged *mergedPCDCamera) Projector(ctx context.Context) (transform.Projector, error) {
	var proj transform.Projector
	return proj, errors.New("Projector is unimplemented")
}

// Stream is a part of the camera interface but is not implemented for replay.
func (merged *mergedPCDCamera) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	var stream gostream.VideoStream
	return stream, errors.New("Stream is unimplemented")
}
