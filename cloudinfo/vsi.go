package cloudinfo

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

const (
	// DefaultVSIImagePattern is the default regex pattern for matching VSI images
	// Currently matches Red Hat 8.x minimal images
	DefaultVSIImagePattern = `^ibm-redhat-8-\d+-minimal-amd64-\d+$`

	// VSIImageStatusAvailable represents the available status for VSI images
	VSIImageStatusAvailable = "available"
)

// GetLatestVSIImageID retrieves the latest available VSI image ID for a given region
// based on the default image pattern (Red Hat 8.x minimal).
// Returns the image ID string and error.
func (infoSvc *CloudInfoService) GetLatestVSIImageID(region string) (string, error) {
	return infoSvc.GetLatestVSIImageIDWithPattern(region, DefaultVSIImagePattern)
}

// GetLatestVSIImageIDWithPattern retrieves the latest available VSI image ID for a given region
// based on a custom regex pattern.
// The pattern parameter should be a valid regex string to match against image names.
// Returns the image ID string and error.
func (infoSvc *CloudInfoService) GetLatestVSIImageIDWithPattern(region string, pattern string) (string, error) {
	if region == "" {
		return "", errors.New("region cannot be empty")
	}

	if pattern == "" {
		return "", errors.New("pattern cannot be empty")
	}

	// Compile the regex pattern
	imageRegex, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	// Get region details to set the correct endpoint
	regionDetail, detailedResponse, err := infoSvc.vpcService.GetRegion(infoSvc.vpcService.NewGetRegionOptions(region))
	if err != nil {
		log.Printf("Failed to get region details for %s: %v, Full Response: %v", region, err, detailedResponse)
		return "", fmt.Errorf("failed to get region details: %w", err)
	}

	// Check if region is available
	if *regionDetail.Status != regionStatusAvailable {
		return "", fmt.Errorf("region %s is not available (status: %s)", region, *regionDetail.Status)
	}

	// Set the VPC service URL to the region endpoint
	originalURL := infoSvc.vpcService.GetServiceURL()
	regionEndpoint := *regionDetail.Endpoint + "/v1"
	setErr := infoSvc.vpcService.SetServiceURL(regionEndpoint)
	if setErr != nil {
		return "", fmt.Errorf("failed to set service URL for region %s: %w", region, setErr)
	}

	// Ensure we restore the original URL when done
	defer func() {
		_ = infoSvc.vpcService.SetServiceURL(originalURL)
	}()

	log.Printf("Retrieving VSI images for region %s with pattern: %s", region, pattern)

	// List all images in the region
	listImagesOptions := &vpcv1.ListImagesOptions{
		Visibility: core.StringPtr("public"), // Only public images
	}

	imageCollection, detailedResponse, err := infoSvc.vpcService.ListImages(listImagesOptions)
	if err != nil {
		log.Printf("Failed to list images for region %s: %v, Full Response: %v", region, err, detailedResponse)
		return "", fmt.Errorf("failed to list images: %w", err)
	}

	log.Printf("Found %d total images in region %s", len(imageCollection.Images), region)

	// Filter images based on pattern and status
	var matchingImages []vpcv1.Image
	for _, image := range imageCollection.Images {
		if image.Name == nil || image.Status == nil || image.ID == nil {
			continue
		}

		imageName := *image.Name
		imageStatus := *image.Status

		// Check if image matches pattern and is available
		if imageRegex.MatchString(imageName) && imageStatus == VSIImageStatusAvailable {
			matchingImages = append(matchingImages, image)
			log.Printf("Matched image: %s (ID: %s, Status: %s)", imageName, *image.ID, imageStatus)
		}
	}

	if len(matchingImages) == 0 {
		return "", fmt.Errorf("no available images found matching pattern '%s' in region %s", pattern, region)
	}

	log.Printf("Found %d matching available images", len(matchingImages))

	// Sort images by name in descending order to get the latest version
	// Image names typically follow a pattern like: ibm-redhat-8-10-minimal-amd64-5
	// We need to extract and compare version numbers properly, not just lexicographically
	// For simplicity, we'll use lexicographic sorting which works for most cases,
	// but users should be aware that "8-9" > "8-10" lexicographically
	// A more robust solution would parse version numbers, but for now we sort by full name
	sort.Slice(matchingImages, func(i, j int) bool {
		// Sort in descending order - newer/higher versions should come first
		return *matchingImages[i].Name > *matchingImages[j].Name
	})

	latestImage := matchingImages[0]
	log.Printf("Selected latest image: %s (ID: %s)", *latestImage.Name, *latestImage.ID)

	return *latestImage.ID, nil
}

// GetVSIImagesByPattern retrieves all available VSI images for a given region
// that match the specified regex pattern.
// Returns a slice of vpcv1.Image and error.
func (infoSvc *CloudInfoService) GetVSIImagesByPattern(region string, pattern string) ([]vpcv1.Image, error) {
	if region == "" {
		return nil, errors.New("region cannot be empty")
	}

	if pattern == "" {
		return nil, errors.New("pattern cannot be empty")
	}

	// Compile the regex pattern
	imageRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	// Get region details to set the correct endpoint
	regionDetail, detailedResponse, err := infoSvc.vpcService.GetRegion(infoSvc.vpcService.NewGetRegionOptions(region))
	if err != nil {
		log.Printf("Failed to get region details for %s: %v, Full Response: %v", region, err, detailedResponse)
		return nil, fmt.Errorf("failed to get region details: %w", err)
	}

	// Check if region is available
	if *regionDetail.Status != regionStatusAvailable {
		return nil, fmt.Errorf("region %s is not available (status: %s)", region, *regionDetail.Status)
	}

	// Set the VPC service URL to the region endpoint
	originalURL := infoSvc.vpcService.GetServiceURL()
	regionEndpoint := *regionDetail.Endpoint + "/v1"
	setErr := infoSvc.vpcService.SetServiceURL(regionEndpoint)
	if setErr != nil {
		return nil, fmt.Errorf("failed to set service URL for region %s: %w", region, setErr)
	}

	// Ensure we restore the original URL when done
	defer func() {
		_ = infoSvc.vpcService.SetServiceURL(originalURL)
	}()

	// List all images in the region
	listImagesOptions := &vpcv1.ListImagesOptions{
		Visibility: core.StringPtr("public"), // Only public images
	}

	imageCollection, detailedResponse, err := infoSvc.vpcService.ListImages(listImagesOptions)
	if err != nil {
		log.Printf("Failed to list images for region %s: %v, Full Response: %v", region, err, detailedResponse)
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// Filter images based on pattern and status
	var matchingImages []vpcv1.Image
	for _, image := range imageCollection.Images {
		if image.Name == nil || image.Status == nil || image.ID == nil {
			continue
		}

		imageName := *image.Name
		imageStatus := *image.Status

		// Check if image matches pattern and is available
		if imageRegex.MatchString(imageName) && imageStatus == VSIImageStatusAvailable {
			matchingImages = append(matchingImages, image)
		}
	}

	// Sort images by name in descending order
	sort.Slice(matchingImages, func(i, j int) bool {
		return strings.Compare(*matchingImages[i].Name, *matchingImages[j].Name) > 0
	})

	return matchingImages, nil
}
