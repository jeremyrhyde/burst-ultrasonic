// Package main provides an implementation for an ultrasonic sensor wrapped as a camera
package main

import (
	"context"
	"errors"
	"image"
	"math/rand"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/sensor"
	ultrasense "go.viam.com/rdk/components/sensor/ultrasonic"
	"go.viam.com/rdk/logging"
	pointcloud "go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
)

var model = resource.DefaultModelFamily.WithModel("burst_ultrasonic")

type ultrasonicWrapper struct {
	usSensor          sensor.Sensor
	StandardDeviation float64
	NumPoints         int
}

type Config struct {
	TriggerPin    string  `json:"trigger_pin"`
	EchoInterrupt string  `json:"echo_interrupt_pin"`
	Board         string  `json:"board"`
	TimeoutMs     uint    `json:"timeout_ms,omitempty"`
	StandardDev   float64 `json:"st_dev,omitempty"`
	NumPoints     uint    `json:"num_points,omitempty"`
}

func init() {
	resource.RegisterComponent(
		camera.API,
		model,
		resource.Registration[camera.Camera, *ultrasense.Config]{
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

func newCamera(ctx context.Context, deps resource.Dependencies, name resource.Name,
	newConf *Config, logger logging.Logger,
) (camera.Camera, error) {

	ultrasenseConfig := &ultrasense.Config{
		TriggerPin:    newConf.TriggerPin,
		EchoInterrupt: newConf.EchoInterrupt,
		Board:         newConf.Board,
		TimeoutMs:     newConf.TimeoutMs,
	}

	usSensor, err := ultrasense.NewSensor(ctx, deps, name, ultrasenseConfig, logger)
	if err != nil {
		return nil, err
	}

	return cameraFromSensor(ctx, name, usSensor, newConf.StandardDev, int(newConf.NumPoints), logger)
}

func cameraFromSensor(ctx context.Context, name resource.Name, usSensor sensor.Sensor, stdev float64, numpoints int, logger logging.Logger) (camera.Camera, error) {
	usWrapper := ultrasonicWrapper{
		usSensor:          usSensor,
		StandardDeviation: stdev,
		NumPoints:         numpoints,
	}

	usVideoSource, err := camera.NewVideoSourceFromReader(ctx, &usWrapper, nil, camera.UnspecifiedStream)
	if err != nil {
		return nil, err
	}

	return camera.FromVideoSource(name, usVideoSource, logger), nil
}

// NextPointCloud queries the ultrasonic sensor then returns the result as a pointcloud,
// with a single point at (0, 0, distance).
func (bultra *ultrasonicWrapper) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	readings, err := bultra.usSensor.Readings(ctx, nil)
	if err != nil {
		return nil, err
	}
	distFloat, ok := readings["distance"].(float64)
	if !ok {
		return nil, errors.New("unable to convert distance to float64")
	}

	return bultra.burst(distFloat * 1000)
}

// Properties returns the properties of the ultrasonic camera.
func (bultra *ultrasonicWrapper) Properties(ctx context.Context) (camera.Properties, error) {
	return camera.Properties{
		SupportsPCD: true,
		ImageType:   camera.UnspecifiedStream,
	}, nil
}

// Close closes the underlying ultrasonic sensor and the camera itself.
func (bultra *ultrasonicWrapper) Close(ctx context.Context) error {
	err := bultra.usSensor.Close(ctx)
	return err
}

// Read returns a not yet implemented error, as it is not needed for the ultrasonic camera.
func (bultra *ultrasonicWrapper) Read(ctx context.Context) (image.Image, func(), error) {
	return nil, nil, errors.New("not yet implemented")
}

func (bultra *ultrasonicWrapper) burst(dist float64) (pointcloud.PointCloud, error) {
	pc := pointcloud.New()
	basicData := pointcloud.NewBasicData()

	for i := 0; i < bultra.NumPoints; i++ {
		y := rand.NormFloat64() * bultra.StandardDeviation
		z := rand.NormFloat64()*bultra.StandardDeviation + dist

		if err := pc.Set(r3.Vector{X: 0, Y: y, Z: z}, basicData); err != nil {
			return nil, err
		}
	}

	return pc, nil
}
