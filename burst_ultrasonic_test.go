package main

import (
	"context"
	"testing"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	pointcloud "go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/testutils/inject"
	"go.viam.com/test"
)

func createMockCamera(name string, points []r3.Vector) camera.Camera {
	pc := pointcloud.New()
	for _, pt := range points {
		pc.Set(pt, pointcloud.NewBasicData())
	}

	cam := inject.NewCamera(name)
	cam.NextPointCloudFunc = func(ctx context.Context) (pointcloud.PointCloud, error) {
		return pc, nil
	}

	return cam
}

func TestNextPointCloud(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	camName := "cam"
	points := []r3.Vector{{X: 0, Y: 0, Z: 1000}}
	cam := createMockCamera(camName, points)

	burstUltra := &burstUltrasonic{
		usSensor:          cam,
		StandardDeviation: 5,
		NumPoints:         10000,
		logger:            logger,
	}

	pc, err := burstUltra.NextPointCloud(ctx)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, pc.Size(), test.ShouldEqual, burstUltra.NumPoints)

	xSum, ySum, zSum := 0., 0., 0.
	pc.Iterate(0, 0, func(p r3.Vector, d pointcloud.Data) bool {
		xSum += p.X
		ySum += p.Y
		zSum += p.Z
		return true
	})
	test.That(t, xSum/float64(burstUltra.NumPoints), test.ShouldBeBetweenOrEqual,
		points[0].X-burstUltra.StandardDeviation,
		points[0].X+burstUltra.StandardDeviation,
	)
}
