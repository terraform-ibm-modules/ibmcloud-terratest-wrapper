package cloudinfo

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/assert"
)

// mockVpcServiceForImages extends vpcServiceMock with custom image listing behavior
type mockVpcServiceForImages struct {
	vpcServiceMock
	images []vpcv1.Image
}

func (m *mockVpcServiceForImages) ListImages(options *vpcv1.ListImagesOptions) (*vpcv1.ImageCollection, *core.DetailedResponse, error) {
	return &vpcv1.ImageCollection{Images: m.images}, nil, nil
}

func TestGetLatestVSIImageID(t *testing.T) {
	t.Run("Success - Returns latest Red Hat image", func(t *testing.T) {
		// Create mock images
		// Note: Lexicographic sorting means "8-9" > "8-10" > "8-8"
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-redhat-8-10-minimal-amd64-5"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		image3Name := "ibm-redhat-8-9-minimal-amd64-4"
		image3ID := "r006-11111111-2222-3333-4444-555555555555"
		image3Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			{Name: &image3Name, ID: &image3ID, Status: &image3Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		imageID, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.NoError(t, err)
		// Lexicographically, "8-9" > "8-10", so 8-9 is selected
		assert.Equal(t, image3ID, imageID, "Should return the lexicographically latest image (8-9)")
	})

	t.Run("Error - Empty region", func(t *testing.T) {
		infoSvc := &CloudInfoService{}
		_, err := infoSvc.GetLatestVSIImageID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region cannot be empty")
	})

	t.Run("Error - No matching images", func(t *testing.T) {
		// Return images that don't match the pattern
		imageName := "ibm-ubuntu-20-04-minimal-amd64-1"
		imageID := "r006-12345678-1234-1234-1234-123456789abc"
		imageStatus := "available"

		mockVpc := &mockVpcServiceForImages{
			images: []vpcv1.Image{
				{Name: &imageName, ID: &imageID, Status: &imageStatus},
			},
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		_, err := infoSvc.GetLatestVSIImageID("us-south")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no available images found")
	})

	t.Run("Success - Filters out deprecated images", func(t *testing.T) {
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "deprecated"

		image2Name := "ibm-redhat-8-10-minimal-amd64-5"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		imageID, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.NoError(t, err)
		assert.Equal(t, image2ID, imageID, "Should skip deprecated image and return available one")
	})

	t.Run("Success - Filters out obsolete images", func(t *testing.T) {
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "obsolete"

		image2Name := "ibm-redhat-8-10-minimal-amd64-5"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		imageID, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.NoError(t, err)
		assert.Equal(t, image2ID, imageID, "Should skip obsolete image and return available one")
	})
}

func TestGetLatestVSIImageIDWithPattern(t *testing.T) {
	t.Run("Success - Custom pattern for Ubuntu", func(t *testing.T) {
		image1Name := "ibm-ubuntu-20-04-minimal-amd64-1"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-ubuntu-22-04-minimal-amd64-2"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		// Custom pattern for Ubuntu images
		pattern := `^ibm-ubuntu-\d+-\d+-minimal-amd64-\d+$`
		imageID, err := infoSvc.GetLatestVSIImageIDWithPattern("us-south", pattern)

		assert.NoError(t, err)
		assert.Equal(t, image2ID, imageID, "Should return the latest Ubuntu image (22-04)")
	})

	t.Run("Error - Invalid regex pattern", func(t *testing.T) {
		mockVpc := &mockVpcServiceForImages{}
		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		_, err := infoSvc.GetLatestVSIImageIDWithPattern("us-south", "[invalid(regex")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid regex pattern")
	})

	t.Run("Error - Empty pattern", func(t *testing.T) {
		infoSvc := &CloudInfoService{}
		_, err := infoSvc.GetLatestVSIImageIDWithPattern("us-south", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern cannot be empty")
	})
}

func TestGetVSIImagesByPattern(t *testing.T) {
	t.Run("Success - Returns all matching images sorted", func(t *testing.T) {
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-redhat-8-10-minimal-amd64-5"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		image3Name := "ibm-redhat-8-9-minimal-amd64-4"
		image3ID := "r006-11111111-2222-3333-4444-555555555555"
		image3Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			{Name: &image3Name, ID: &image3ID, Status: &image3Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)

		assert.NoError(t, err)
		assert.Len(t, images, 3)
		// Should be sorted in descending lexicographic order: 8-9 > 8-8 > 8-10
		assert.Equal(t, image3Name, *images[0].Name)
		assert.Equal(t, image1Name, *images[1].Name)
		assert.Equal(t, image2Name, *images[2].Name)
	})

	t.Run("Success - Filters by pattern", func(t *testing.T) {
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-ubuntu-20-04-minimal-amd64-1"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		mockImages := []vpcv1.Image{
			{Name: &image1Name, ID: &image1ID, Status: &image1Status},
			{Name: &image2Name, ID: &image2ID, Status: &image2Status},
		}

		mockVpc := &mockVpcServiceForImages{
			images: mockImages,
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		// Only Red Hat images should match
		images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)

		assert.NoError(t, err)
		assert.Len(t, images, 1)
		assert.Equal(t, image1Name, *images[0].Name)
	})
}

func TestDefaultVSIImagePattern(t *testing.T) {
	t.Run("Pattern matches expected Red Hat images", func(t *testing.T) {
		validNames := []string{
			"ibm-redhat-8-8-minimal-amd64-3",
			"ibm-redhat-8-10-minimal-amd64-5",
			"ibm-redhat-8-9-minimal-amd64-4",
			"ibm-redhat-8-12-minimal-amd64-1",
		}

		invalidNames := []string{
			"ibm-redhat-7-8-minimal-amd64-3",   // Wrong major version
			"ibm-ubuntu-20-04-minimal-amd64-1", // Wrong OS
			"ibm-redhat-8-8-full-amd64-3",      // Not minimal
			"ibm-redhat-8-8-minimal-s390x-3",   // Wrong architecture
			"redhat-8-8-minimal-amd64-3",       // Missing ibm prefix
		}

		for _, name := range validNames {
			status := "available"
			id := "r006-test-id"
			image := vpcv1.Image{Name: &name, Status: &status, ID: &id}
			mockVpc := &mockVpcServiceForImages{
				images: []vpcv1.Image{image},
			}
			infoSvc := &CloudInfoService{vpcService: mockVpc}

			images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)
			assert.NoError(t, err)
			assert.Len(t, images, 1, "Pattern should match: %s", name)
		}

		for _, name := range invalidNames {
			status := "available"
			id := "r006-test-id"
			image := vpcv1.Image{Name: &name, Status: &status, ID: &id}
			mockVpc := &mockVpcServiceForImages{
				images: []vpcv1.Image{image},
			}
			infoSvc := &CloudInfoService{vpcService: mockVpc}

			images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)
			assert.NoError(t, err)
			assert.Len(t, images, 0, "Pattern should NOT match: %s", name)
		}
	})
}
