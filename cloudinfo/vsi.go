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
	// DefaultVSIImagePattern matches Red Hat 8.x minimal amd64 images.
	DefaultVSIImagePattern = `^ibm-redhat-8-\d+-minimal-amd64-\d+$`

	// VSIImageStatusAvailable is the only image status safe to use in tests.
	VSIImageStatusAvailable = "available"
)

// listAllPublicImages fetches all pages of public images from the VPC API.
// The API returns at most 50 images per page; this follows Next.Href until exhausted.
func (infoSvc *CloudInfoService) listAllPublicImages() ([]vpcv1.Image, error) {
	var allImages []vpcv1.Image

	opts := &vpcv1.ListImagesOptions{
		Visibility: core.StringPtr("public"),
	}

	for {
		collection, detailedResponse, err := infoSvc.vpcService.ListImages(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list images: %w (response: %v)", err, detailedResponse)
		}

		allImages = append(allImages, collection.Images...)

		if collection.Next == nil || collection.Next.Href == nil {
			break
		}

		start, parseErr := core.GetQueryParam(collection.Next.Href, "start")
		if parseErr != nil || start == nil {
			break
		}
		opts.Start = start
	}

	return allImages, nil
}

// setRegionEndpoint points the VPC service at the given region's endpoint.
// Returns a restore function that resets the URL; call it with defer.
func (infoSvc *CloudInfoService) setRegionEndpoint(region string) (func(), error) {
	regionDetail, detailedResponse, err := infoSvc.vpcService.GetRegion(infoSvc.vpcService.NewGetRegionOptions(region))
	if err != nil {
		log.Printf("Failed to get region details for %s: %v, Full Response: %v", region, err, detailedResponse)
		return nil, fmt.Errorf("failed to get region details: %w", err)
	}

	if *regionDetail.Status != regionStatusAvailable {
		return nil, fmt.Errorf("region %s is not available (status: %s)", region, *regionDetail.Status)
	}

	originalURL := infoSvc.vpcService.GetServiceURL()
	regionEndpoint := *regionDetail.Endpoint + "/v1"
	if setErr := infoSvc.vpcService.SetServiceURL(regionEndpoint); setErr != nil {
		return nil, fmt.Errorf("failed to set service URL for region %s: %w", region, setErr)
	}

	return func() { _ = infoSvc.vpcService.SetServiceURL(originalURL) }, nil
}

// GetLatestVSIImageID returns the latest available Red Hat 8.x minimal image ID for the region.
// Use this instead of hard-coding image IDs in tests to avoid region lock-in and deprecated images.
func (infoSvc *CloudInfoService) GetLatestVSIImageID(region string) (string, error) {
	return infoSvc.GetLatestVSIImageIDWithPattern(region, DefaultVSIImagePattern)
}

// GetLatestVSIImageIDWithPattern returns the latest available image ID matching the given regex pattern.
// Fetches all pages of public images, filters by pattern and "available" status, returns the newest name.
func (infoSvc *CloudInfoService) GetLatestVSIImageIDWithPattern(region string, pattern string) (string, error) {
	if region == "" {
		return "", errors.New("region cannot be empty")
	}
	if pattern == "" {
		return "", errors.New("pattern cannot be empty")
	}

	imageRegex, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	restore, err := infoSvc.setRegionEndpoint(region)
	if err != nil {
		return "", err
	}
	defer restore()

	log.Printf("Retrieving VSI images for region %s with pattern: %s", region, pattern)

	allImages, err := infoSvc.listAllPublicImages()
	if err != nil {
		log.Printf("Failed to list images for region %s: %v", region, err)
		return "", err
	}

	log.Printf("Found %d total images in region %s", len(allImages), region)

	var matchingImages []vpcv1.Image
	for _, image := range allImages {
		if image.Name == nil || image.Status == nil || image.ID == nil {
			continue
		}
		if imageRegex.MatchString(*image.Name) && *image.Status == VSIImageStatusAvailable {
			matchingImages = append(matchingImages, image)
			log.Printf("Matched image: %s (ID: %s, Status: %s)", *image.Name, *image.ID, *image.Status)
		}
	}

	if len(matchingImages) == 0 {
		return "", fmt.Errorf("no available images found matching pattern '%s' in region %s", pattern, region)
	}

	log.Printf("Found %d matching available images", len(matchingImages))

	sort.Slice(matchingImages, func(i, j int) bool {
		return *matchingImages[i].Name > *matchingImages[j].Name
	})

	latestImage := matchingImages[0]
	log.Printf("Selected latest image: %s (ID: %s)", *latestImage.Name, *latestImage.ID)

	return *latestImage.ID, nil
}

// GetVSIImagesByPattern returns all available images matching the regex pattern, sorted newest first.
// Use GetLatestVSIImageID when only one image ID is needed.
func (infoSvc *CloudInfoService) GetVSIImagesByPattern(region string, pattern string) ([]vpcv1.Image, error) {
	if region == "" {
		return nil, errors.New("region cannot be empty")
	}
	if pattern == "" {
		return nil, errors.New("pattern cannot be empty")
	}

	imageRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	restore, err := infoSvc.setRegionEndpoint(region)
	if err != nil {
		return nil, err
	}
	defer restore()

	allImages, err := infoSvc.listAllPublicImages()
	if err != nil {
		log.Printf("Failed to list images for region %s: %v", region, err)
		return nil, err
	}

	var matchingImages []vpcv1.Image
	for _, image := range allImages {
		if image.Name == nil || image.Status == nil || image.ID == nil {
			continue
		}
		if imageRegex.MatchString(*image.Name) && *image.Status == VSIImageStatusAvailable {
			matchingImages = append(matchingImages, image)
		}
	}

	sort.Slice(matchingImages, func(i, j int) bool {
		return strings.Compare(*matchingImages[i].Name, *matchingImages[j].Name) > 0
	})

	return matchingImages, nil
}
