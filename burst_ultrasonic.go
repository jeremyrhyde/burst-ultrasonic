// Package main provides an implementation for an ultrasonic sensor wrapped as a camera
package main

import (
	"context"
	"image"
	"math/rand"

	"github.com/pkg/errors"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	pointcloud "go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
)

var model = resource.NewModel("viam", "camera", "burst_ultrasonic")

type burstUltrasonic struct {
	resource.Named
	usSensor          camera.Camera
	StandardDeviation float64
	NumPoints         int
	logger            logging.Logger
}

type Config struct {
	UltrasonicSensor string  `json:"ultrasonic_sensor,omitempty"`
	StandardDev      float64 `json:"st_dev,omitempty"`
	NumPoints        uint    `json:"num_points,omitempty"`
}

func init() {
	resource.RegisterComponent(
		camera.API,
		model,
		resource.Registration[camera.Camera, *Config]{
			Constructor: func(
				ctx context.Context,
				deps resource.Dependencies,
				conf resource.Config,
				logger logging.Logger,
			) (camera.Camera, error) {
				newConf, err := resource.NativeConfig[*Config](conf)
				if err != nil {
					return nil, err
				}

				return newCamera(ctx, deps, conf.ResourceName(), newConf, logger)
			},
		})
}

// Validate ensures all parts of the config are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	var deps []string

	if conf.UltrasonicSensor == "" {
		return nil, resource.NewConfigValidationFieldRequiredError(path, "ultrasonic_sensor")
	}

	deps = append(deps, conf.UltrasonicSensor)
	return deps, nil
}

func newCamera(ctx context.Context, deps resource.Dependencies, name resource.Name,
	newConf *Config, logger logging.Logger,
) (camera.Camera, error) {
	ultrasonic, err := camera.FromDependencies(deps, newConf.UltrasonicSensor)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting camera %v", newConf.UltrasonicSensor)
	}

	numPoints := int(newConf.NumPoints)
	if newConf.NumPoints == 1 {
		numPoints = 1
	}

	stDev := newConf.StandardDev
	if stDev < 0 {
		return nil, errors.New("Standard deviation must be able 0")
	}

	burstUltra := &burstUltrasonic{
		usSensor:          ultrasonic,
		StandardDeviation: newConf.StandardDev,
		NumPoints:         numPoints,
		logger:            logger,
	}

	return burstUltra, nil
}

func (burstUltra *burstUltrasonic) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	ultrasonic, err := camera.FromDependencies(deps, newConf.UltrasonicSensor)
	if err != nil {
		return errors.Wrapf(err, "error getting camera %v", newConf.UltrasonicSensor)
	}

	burstUltra.usSensor = ultrasonic

	return nil
}

// NextPointCloud queries the ultrasonic sensor then returns the result as a pointcloud,
// with a single point at (0, 0, distance).
func (burstUltra *burstUltrasonic) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	pc, err := burstUltra.usSensor.NextPointCloud(ctx)
	if err != nil {
		burstUltra.logger.Error("issue reading sensor")
		return nil, err
	}

	return burstUltra.burst(pc)
}

// Properties returns the properties of the ultrasonic camera.
func (burstUltra *burstUltrasonic) Properties(ctx context.Context) (camera.Properties, error) {
	return camera.Properties{
		SupportsPCD: true,
		ImageType:   camera.UnspecifiedStream,
	}, nil
}

// Close closes the underlying ultrasonic sensor and the camera itself.
func (burstUltra *burstUltrasonic) Close(ctx context.Context) error {
	err := burstUltra.usSensor.Close(ctx)
	return err
}

func (burstUltra *burstUltrasonic) burst(pc pointcloud.PointCloud) (pointcloud.PointCloud, error) {
	pcReturn := pointcloud.New()
	basicData := pointcloud.NewBasicData()

	pc.Iterate(0, 0, func(p r3.Vector, d pointcloud.Data) bool {
		if err := pcReturn.Set(r3.Vector{X: 0, Y: 0, Z: p.Z}, basicData); err != nil {
			return false
		}

		for i := 0; i < burstUltra.NumPoints-1; i++ {
			y := rand.NormFloat64() * burstUltra.StandardDeviation
			z := rand.NormFloat64()*burstUltra.StandardDeviation + p.Z

			if err := pcReturn.Set(r3.Vector{X: 0, Y: y, Z: z}, basicData); err != nil {
				return false
			}
		}
		return true
	})

	return pcReturn, nil
}

func (bultra *burstUltrasonic) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("unimplemented")
}

func (bultra *burstUltrasonic) Images(context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	return nil, resource.ResponseMetadata{}, errors.New("unimplemented")
}

func (bultra *burstUltrasonic) Projector(context.Context) (transform.Projector, error) {
	return nil, errors.New("unimplemented")
}

func (bultra *burstUltrasonic) Stream(context.Context, ...gostream.ErrorHandler) (gostream.MediaStream[image.Image], error) {
	return nil, errors.New("unimplemented")
}
