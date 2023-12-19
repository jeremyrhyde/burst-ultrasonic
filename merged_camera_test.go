// Package main merges the the result of NextPointCloud from multiple cameras
package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/robot/framesystem"
	"go.viam.com/rdk/spatialmath"
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

// createCameraLink instantiates the camera to base link for the frame system.
func createCameraLink(camName, baseFrame string) (*referenceframe.LinkInFrame, error) {
	camPose := spatialmath.NewPoseFromPoint(r3.Vector{X: 0, Y: 0, Z: 0})
	camSphere, err := spatialmath.NewSphere(camPose, 5, "cam-sphere")
	if err != nil {
		return nil, err
	}

	camLink := referenceframe.NewLinkInFrame(
		baseFrame,
		spatialmath.NewZeroPose(),
		camName,
		camSphere,
	)
	return camLink, nil
}

// createFrameSystemService will create a basic frame service from the list of parts.
func createFrameSystemService(
	ctx context.Context,
	cameras []camera.Camera,
	logger logging.Logger,
) (framesystem.Service, error) {
	var fsParts []*referenceframe.FrameSystemPart
	deps := make(resource.Dependencies)

	// create camera link
	for _, cam := range cameras {
		cameraLink, err := createCameraLink(cam.Name().Name, "world")
		if err != nil {
			return nil, err
		}
		fsParts = append(fsParts, &referenceframe.FrameSystemPart{FrameConfig: cameraLink})
		deps[cam.Name()] = cam
	}

	fsSvc, err := framesystem.New(ctx, deps, logger)
	if err != nil {
		return nil, err
	}
	conf := resource.Config{
		ConvertedAttributes: &framesystem.Config{Parts: fsParts},
	}
	if err := fsSvc.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}

	return fsSvc, nil
}

func TestMergedCamera(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	camName1 := "cam1"
	points1 := []r3.Vector{{X: 0, Y: 1, Z: 2}}
	cam1 := createMockCamera(camName1, points1)

	camName2 := "cam2"
	points2 := []r3.Vector{{X: 0, Y: 0, Z: 2}}
	cam2 := createMockCamera(camName2, points2)

	allPoints := append(points1, points2...)
	cameras := []camera.Camera{cam1, cam2}

	fmt.Println(cameras)

	fsService, err := createFrameSystemService(ctx, cameras, logger)

	// Create merged camera struct
	mergedCam := mergedCamera{
		cameras:   cameras,
		fsService: fsService,
		logger:    logger,
	}
	fmt.Println("sdsdsaaaaaaa")
	pc, err := mergedCam.NextPointCloud(ctx)
	fmt.Println("sdsdsdsdsdsdsdss")
	test.That(t, err, test.ShouldBeNil)
	fmt.Println("sdsdsd122211s")
	fmt.Println(pc)
	test.That(t, pc.Size(), test.ShouldEqual, 2)

	fmt.Println("sdsdsds")
	pc.Iterate(0, 0, func(p r3.Vector, d pointcloud.Data) bool {
		found := false
		for _, pt := range allPoints {
			if p == pt {
				found = true
			}
		}
		test.That(t, found, test.ShouldBeTrue)
		return true
	})

}
