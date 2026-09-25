package cloudinfo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/assert"
)

// mockVpcServiceForImages supports multi-page responses to test pagination in listAllPublicImages.
// Each element of pages is one API page; pageIndex advances on each ListImages call.
type mockVpcServiceForImages struct {
	vpcServiceMock
	pages     [][]vpcv1.Image // each element is one page of images
	pageIndex int             // tracks the current page
}

// ListImages returns the next page and sets collection.Next if more pages remain.
func (m *mockVpcServiceForImages) ListImages(options *vpcv1.ListImagesOptions) (*vpcv1.ImageCollection, *core.DetailedResponse, error) {
	if len(m.pages) == 0 {
		return &vpcv1.ImageCollection{Images: []vpcv1.Image{}}, nil, nil
	}

	idx := m.pageIndex
	if idx >= len(m.pages) {
		idx = len(m.pages) - 1
	}
	m.pageIndex++

	images := m.pages[idx]
	collection := &vpcv1.ImageCollection{Images: images}

	// Set Next so listAllPublicImages knows to fetch the next page.
	if idx < len(m.pages)-1 {
		nextHref := fmt.Sprintf("https://mock-vpc/v1/images?start=page%d&limit=50", idx+1)
		collection.Next = &vpcv1.ImageCollectionNext{Href: &nextHref}
	}

	return collection, nil, nil
}

func TestGetLatestVSIImageID(t *testing.T) {
	t.Run("Success - Returns latest Red Hat image", func(t *testing.T) {
		image1Name := "ibm-redhat-8-8-minimal-amd64-3"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-redhat-8-10-minimal-amd64-5"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		image3Name := "ibm-redhat-8-9-minimal-amd64-4"
		image3ID := "r006-11111111-2222-3333-4444-555555555555"
		image3Status := "available"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
				{Name: &image3Name, ID: &image3ID, Status: &image3Status},
			}},
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
			pages: [][]vpcv1.Image{{{Name: &imageName, ID: &imageID, Status: &imageStatus}}},
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

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			}},
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

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			}},
		}

		infoSvc := &CloudInfoService{
			vpcService: mockVpc,
		}

		imageID, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.NoError(t, err)
		assert.Equal(t, image2ID, imageID, "Should skip obsolete image and return available one")
	})
}

func TestGetLatestVSIImageIDErrors(t *testing.T) {
	t.Run("Error - GetRegion failure propagates", func(t *testing.T) {
		mockVpc := &mockVpcServiceForImages{}
		mockVpc.shouldFailGetRegion = true
		mockVpc.getRegionError = errors.New("vpc api unavailable")

		infoSvc := &CloudInfoService{vpcService: mockVpc}
		_, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get region details")
	})

	t.Run("Error - Unavailable region is rejected", func(t *testing.T) {
		// Override GetRegion to return a non-available status directly.
		// vpcServiceMock.GetRegion always returns regionStatusAvailable, so we use
		// a small inline mock that returns "unavailable" instead.
		mockVpc := &unavailableRegionMock{}

		infoSvc := &CloudInfoService{vpcService: mockVpc}
		_, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not available")
	})

	t.Run("Error - SetServiceURL failure propagates", func(t *testing.T) {
		mockVpc := &mockVpcServiceForImages{}
		mockVpc.shouldFailSetServiceURL = true

		infoSvc := &CloudInfoService{vpcService: mockVpc}
		_, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set service URL")
	})

	t.Run("Error - ListImages API error propagates", func(t *testing.T) {
		mockVpc := &listImagesErrorMock{}

		infoSvc := &CloudInfoService{vpcService: mockVpc}
		_, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list images")
	})

	t.Run("Success - Skips images with nil Name/Status/ID fields", func(t *testing.T) {
		validName := "ibm-redhat-8-9-minimal-amd64-1"
		validID := "r006-valid-id"
		validStatus := "available"

		// Image with a nil Name — should be silently skipped.
		nilNameStatus := "available"
		nilNameID := "r006-nil-name-id"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: nil, Status: &nilNameStatus, ID: &nilNameID},
				{Name: &validName, ID: &validID, Status: &validStatus},
			}},
		}

		infoSvc := &CloudInfoService{vpcService: mockVpc}
		imageID, err := infoSvc.GetLatestVSIImageID("us-south")

		assert.NoError(t, err)
		assert.Equal(t, validID, imageID, "Should skip nil-Name image and return the valid one")
	})
}

// unavailableRegionMock returns a region whose status is "unavailable".
type unavailableRegionMock struct {
	vpcServiceMock
}

func (m *unavailableRegionMock) GetRegion(options *vpcv1.GetRegionOptions) (*vpcv1.Region, *core.DetailedResponse, error) {
	status := "unavailable"
	region := vpcv1.Region{
		Name:     options.Name,
		Endpoint: options.Name,
		Href:     options.Name,
		Status:   &status,
	}
	return &region, nil, nil
}

// listImagesErrorMock returns an error from ListImages so we can test that path.
type listImagesErrorMock struct {
	vpcServiceMock
}

func (m *listImagesErrorMock) ListImages(options *vpcv1.ListImagesOptions) (*vpcv1.ImageCollection, *core.DetailedResponse, error) {
	return nil, &core.DetailedResponse{StatusCode: 500}, errors.New("upstream API error")
}

func TestGetLatestVSIImageIDWithPattern(t *testing.T) {
	t.Run("Success - Custom pattern for Ubuntu", func(t *testing.T) {
		image1Name := "ibm-ubuntu-20-04-minimal-amd64-1"
		image1ID := "r006-12345678-1234-1234-1234-123456789abc"
		image1Status := "available"

		image2Name := "ibm-ubuntu-22-04-minimal-amd64-2"
		image2ID := "r006-87654321-4321-4321-4321-cba987654321"
		image2Status := "available"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			}},
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

func TestGetVSIImagesByPatternErrors(t *testing.T) {
	t.Run("Error - Empty region", func(t *testing.T) {
		infoSvc := &CloudInfoService{}
		_, err := infoSvc.GetVSIImagesByPattern("", DefaultVSIImagePattern)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region cannot be empty")
	})

	t.Run("Error - Empty pattern", func(t *testing.T) {
		infoSvc := &CloudInfoService{}
		_, err := infoSvc.GetVSIImagesByPattern("us-south", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern cannot be empty")
	})

	t.Run("Error - Invalid regex pattern", func(t *testing.T) {
		mockVpc := &mockVpcServiceForImages{}
		infoSvc := &CloudInfoService{vpcService: mockVpc}
		_, err := infoSvc.GetVSIImagesByPattern("us-south", "[invalid(regex")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid regex pattern")
	})

	t.Run("Success - Returns empty slice when nothing matches", func(t *testing.T) {
		imageName := "ibm-ubuntu-20-04-minimal-amd64-1"
		imageID := "r006-ubuntu-id"
		imageStatus := "available"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{{Name: &imageName, ID: &imageID, Status: &imageStatus}}},
		}
		infoSvc := &CloudInfoService{vpcService: mockVpc}

		images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)
		assert.NoError(t, err)
		assert.Empty(t, images, "Should return empty slice when no images match")
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

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
				{Name: &image3Name, ID: &image3ID, Status: &image3Status},
			}},
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

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{{
				{Name: &image1Name, ID: &image1ID, Status: &image1Status},
				{Name: &image2Name, ID: &image2ID, Status: &image2Status},
			}},
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
				pages: [][]vpcv1.Image{{image}},
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
				pages: [][]vpcv1.Image{{image}},
			}
			infoSvc := &CloudInfoService{vpcService: mockVpc}

			images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)
			assert.NoError(t, err)
			assert.Len(t, images, 0, "Pattern should NOT match: %s", name)
		}
	})
}

// TestListAllPublicImagesPagination verifies that images across multiple pages are all collected.
func TestListAllPublicImagesPagination(t *testing.T) {
	t.Run("Success - Collects images across multiple pages", func(t *testing.T) {
		// Page 1: one non-matching image
		page1Name := "ibm-ubuntu-20-04-minimal-amd64-1"
		page1ID := "r006-page1-id"
		page1Status := "available"

		// Page 2: the matching image — would be missed without pagination
		page2Name := "ibm-redhat-8-10-minimal-amd64-5"
		page2ID := "r006-page2-id"
		page2Status := "available"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{
				{{Name: &page1Name, ID: &page1ID, Status: &page1Status}},
				{{Name: &page2Name, ID: &page2ID, Status: &page2Status}},
			},
		}

		infoSvc := &CloudInfoService{vpcService: mockVpc}

		imageID, err := infoSvc.GetLatestVSIImageID("us-south")
		assert.NoError(t, err)
		assert.Equal(t, page2ID, imageID, "Should find the matching image on page 2")
	})

	t.Run("Success - GetVSIImagesByPattern collects all pages", func(t *testing.T) {
		img1Name := "ibm-redhat-8-8-minimal-amd64-3"
		img1ID := "r006-img1-id"
		img1Status := "available"

		img2Name := "ibm-redhat-8-10-minimal-amd64-5"
		img2ID := "r006-img2-id"
		img2Status := "available"

		mockVpc := &mockVpcServiceForImages{
			pages: [][]vpcv1.Image{
				{{Name: &img1Name, ID: &img1ID, Status: &img1Status}},
				{{Name: &img2Name, ID: &img2ID, Status: &img2Status}},
			},
		}

		infoSvc := &CloudInfoService{vpcService: mockVpc}

		images, err := infoSvc.GetVSIImagesByPattern("us-south", DefaultVSIImagePattern)
		assert.NoError(t, err)
		assert.Len(t, images, 2, "Should collect matching images from both pages")
	})
}
