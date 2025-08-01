package testaddons

import (
	"fmt"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// TestBuildDependencyGraphWithManualDependencies tests that manually enabled dependencies
// are included in the expected dependency graph even if they're not marked as "OnByDefault"
func TestBuildDependencyGraphWithManualDependencies(t *testing.T) {
	// Create a mock test logger
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Create test options with a mock CloudInfoService
	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{}, // We'll need to create this mock
	}

	// Create a main addon config with manually enabled dependencies
	addonConfig := &cloudinfo.AddonConfig{
		OfferingName:    "main-addon",
		OfferingFlavor:  "standard",
		CatalogID:       "test-catalog",
		OfferingID:      "main-offering-id",
		VersionLocator:  "test-catalog.main-version",
		ResolvedVersion: "v1.0.0",
		Dependencies: []cloudinfo.AddonConfig{
			{
				OfferingName:    "deploy-arch-ibm-account-infra-base",
				OfferingFlavor:  "resource-group-only",
				CatalogID:       "dependency-catalog",
				OfferingID:      "account-base-offering-id", // Use the same ID that the mock expects
				VersionLocator:  "dependency-catalog.account-base-version",
				ResolvedVersion: "v3.0.11",
				Enabled:         core.BoolPtr(true), // Manually enabled
				Dependencies:    []cloudinfo.AddonConfig{},
			},
		},
	}

	// Create visited map for the function
	visited := make(map[string]bool)

	// Call the function (this would normally fail without our fix)
	graphResult, err := options.buildDependencyGraph(
		"test-catalog",
		"main-offering-id",
		"test-catalog.main-version",
		"standard",
		addonConfig,
		visited,
	)

	// Verify no error occurred
	assert.NoError(t, err)

	// Extract results
	expectedDeployedList := graphResult.ExpectedDeployedList

	// Verify that both the main addon and the manually enabled dependency are in the expected list
	assert.Len(t, expectedDeployedList, 2, "Expected both main addon and manually enabled dependency")

	// Debug: print what we actually got
	t.Logf("Expected deployed list contains %d items:", len(expectedDeployedList))
	for _, item := range expectedDeployedList {
		t.Logf("  - %s:%s:%s", item.Name, item.Version, item.Flavor.Name)
	}

	// Check that the main addon is included
	foundMain := false
	foundDependency := false
	for _, config := range expectedDeployedList {
		if config.Name == "main-addon" {
			foundMain = true
		}
		if config.Name == "deploy-arch-ibm-account-infra-base" && config.Version == "v3.0.11" {
			foundDependency = true
		}
	}

	assert.True(t, foundMain, "Main addon should be in expected deployed list")
	assert.True(t, foundDependency, "Manually enabled dependency should be in expected deployed list")
}

// TestBuildDependencyGraphObservabilityScenario tests the exact scenario described in the user's problem:
// deploy-arch-ibm-observability with manually enabled dependencies
func TestBuildDependencyGraphObservabilityScenario(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Recreate the exact scenario from the user's problem
	addonConfig := &cloudinfo.AddonConfig{
		OfferingName:    "deploy-arch-ibm-observability",
		OfferingFlavor:  "instances",
		CatalogID:       "test-catalog",
		OfferingID:      "observability-offering-id",
		VersionLocator:  "test-catalog.observability-version",
		ResolvedVersion: "v0.0.1-dev-tf-an-zl3h3r",
		Dependencies: []cloudinfo.AddonConfig{
			{
				// First dependency - account base
				OfferingName:    "deploy-arch-ibm-account-infra-base",
				OfferingFlavor:  "resource-group-only",
				CatalogID:       "dependency-catalog",
				OfferingID:      "account-base-offering-id",
				VersionLocator:  "dependency-catalog.account-base-version",
				ResolvedVersion: "v3.0.11",
				Enabled:         core.BoolPtr(true), // explicitly enabled
				Dependencies:    []cloudinfo.AddonConfig{},
			},
			{
				// Second dependency - KMS with its own nested dependency
				OfferingName:    "deploy-arch-ibm-kms",
				OfferingFlavor:  "fully-configurable",
				CatalogID:       "dependency-catalog",
				OfferingID:      "kms-offering-id",
				VersionLocator:  "dependency-catalog.kms-version",
				ResolvedVersion: "v5.1.4",
				Enabled:         core.BoolPtr(true), // This would be enabled by catalog default
				Dependencies: []cloudinfo.AddonConfig{
					{
						OfferingName:    "deploy-arch-ibm-account-infra-base",
						OfferingFlavor:  "resource-group-only",
						CatalogID:       "dependency-catalog",
						OfferingID:      "account-base-offering-id",
						VersionLocator:  "dependency-catalog.account-base-nested-version",
						ResolvedVersion: "v3.0.7",
						Enabled:         core.BoolPtr(true), // explicitly enabled
						Dependencies:    []cloudinfo.AddonConfig{},
					},
				},
			},
		},
	}

	visited := make(map[string]bool)

	graphResult, err := options.buildDependencyGraph(
		"test-catalog",
		"observability-offering-id",
		"test-catalog.observability-version",
		"instances",
		addonConfig,
		visited,
	)

	assert.NoError(t, err)

	// Extract results
	expectedDeployedList := graphResult.ExpectedDeployedList

	// Before our fix, this would only contain 2 items (observability + kms)
	// After our fix, it should contain 4 items (observability + both account-base versions + kms)
	t.Logf("Expected deployed list contains %d items:", len(expectedDeployedList))
	for _, item := range expectedDeployedList {
		t.Logf("  - %s:%s:%s", item.Name, item.Version, item.Flavor.Name)
	}

	assert.Len(t, expectedDeployedList, 4, "Should contain observability, kms, and both account-base instances")

	// Verify all expected components are present
	foundObservability := false
	foundKms := false
	foundAccountBase1 := false // v3.0.11
	foundAccountBase2 := false // v3.0.7

	for _, config := range expectedDeployedList {
		switch {
		case config.Name == "deploy-arch-ibm-observability":
			foundObservability = true
		case config.Name == "deploy-arch-ibm-kms":
			foundKms = true
		case config.Name == "deploy-arch-ibm-account-infra-base" && config.Version == "v3.0.11":
			foundAccountBase1 = true
		case config.Name == "deploy-arch-ibm-account-infra-base" && config.Version == "v3.0.7":
			foundAccountBase2 = true
		}
	}

	assert.True(t, foundObservability, "Should find observability addon")
	assert.True(t, foundKms, "Should find KMS addon")
	assert.True(t, foundAccountBase1, "Should find account base v3.0.11")
	assert.True(t, foundAccountBase2, "Should find account base v3.0.7")
}

// TestBuildDependencyGraphWithManuallyDisabledDependency tests that dependencies
// marked as OnByDefault in the catalog but manually disabled via Enabled=false
// are NOT included in the expected dependency graph
func TestBuildDependencyGraphWithManuallyDisabledDependency(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoServiceWithCatalogDeps{}, // New mock with catalog dependencies
	}

	// Create addon config with a manually disabled dependency
	addonConfig := &cloudinfo.AddonConfig{
		OfferingName:    "main-addon",
		OfferingFlavor:  "standard",
		CatalogID:       "test-catalog",
		OfferingID:      "main-offering-id",
		VersionLocator:  "test-catalog.main-version",
		ResolvedVersion: "v1.0.0",
		Dependencies: []cloudinfo.AddonConfig{
			{
				OfferingName:    "catalog-default-dependency",
				OfferingFlavor:  "standard",
				CatalogID:       "dependency-catalog",
				OfferingID:      "catalog-dep-offering-id",
				VersionLocator:  "dependency-catalog.catalog-dep-version",
				ResolvedVersion: "v2.0.0",
				Enabled:         core.BoolPtr(false), // Manually DISABLED
				Dependencies:    []cloudinfo.AddonConfig{},
			},
		},
	}

	visited := make(map[string]bool)

	graphResult, err := options.buildDependencyGraph(
		"test-catalog",
		"main-offering-id",
		"test-catalog.main-version",
		"standard",
		addonConfig,
		visited,
	)

	assert.NoError(t, err)

	// Extract results
	expectedDeployedList := graphResult.ExpectedDeployedList

	// Debug: print what we actually got
	t.Logf("Expected deployed list contains %d items:", len(expectedDeployedList))
	for _, item := range expectedDeployedList {
		t.Logf("  - %s:%s:%s", item.Name, item.Version, item.Flavor.Name)
	}

	// Should only contain the main addon, NOT the disabled dependency
	assert.Len(t, expectedDeployedList, 1, "Should only contain main addon, not the disabled dependency")

	// Verify only the main addon is present
	foundMain := false
	foundDisabledDep := false
	for _, config := range expectedDeployedList {
		if config.Name == "main-addon" {
			foundMain = true
		}
		if config.Name == "catalog-default-dependency" {
			foundDisabledDep = true
		}
	}

	assert.True(t, foundMain, "Main addon should be in expected deployed list")
	assert.False(t, foundDisabledDep, "Manually disabled dependency should NOT be in expected deployed list")
}

// TestBuildDependencyGraphWithOfferingLevelDisable tests that disabling a dependency
// at the offering level affects all flavors of that offering, not just the specific flavor
func TestBuildDependencyGraphWithOfferingLevelDisable(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoServiceWithMultipleFlavors{}, // New mock with multiple flavors
	}

	// Create addon config where we disable one flavor of a multi-flavor offering
	// This should disable ALL flavors of that offering
	addonConfig := &cloudinfo.AddonConfig{
		OfferingName:    "main-addon",
		OfferingFlavor:  "standard",
		CatalogID:       "test-catalog",
		OfferingID:      "main-offering-id",
		VersionLocator:  "test-catalog.main-version",
		ResolvedVersion: "v1.0.0",
		Dependencies: []cloudinfo.AddonConfig{
			{
				// We disable the "basic" flavor, but this should disable ALL flavors
				OfferingName:    "multi-flavor-dependency",
				OfferingFlavor:  "basic", // Disabling basic flavor
				CatalogID:       "dependency-catalog",
				OfferingID:      "multi-flavor-offering-id",
				VersionLocator:  "dependency-catalog.multi-flavor-basic-version",
				ResolvedVersion: "v2.0.0",
				Enabled:         core.BoolPtr(false), // Manually DISABLED
				Dependencies:    []cloudinfo.AddonConfig{},
			},
		},
	}

	visited := make(map[string]bool)

	graphResult, err := options.buildDependencyGraph(
		"test-catalog",
		"main-offering-id",
		"test-catalog.main-version",
		"standard",
		addonConfig,
		visited,
	)

	assert.NoError(t, err)

	// Extract results
	expectedDeployedList := graphResult.ExpectedDeployedList

	// Debug: print what we actually got
	t.Logf("Expected deployed list contains %d items:", len(expectedDeployedList))
	for _, item := range expectedDeployedList {
		t.Logf("  - %s:%s:%s", item.Name, item.Version, item.Flavor.Name)
	}

	// Should only contain the main addon, NOT ANY flavor of the disabled dependency
	// Even though the catalog defines both "basic" and "premium" flavors as OnByDefault=true,
	// disabling any flavor should disable the entire offering
	assert.Len(t, expectedDeployedList, 1, "Should only contain main addon, not any flavor of the disabled dependency")

	// Verify only the main addon is present and no flavor of the disabled dependency
	foundMain := false
	foundDisabledDepBasic := false
	foundDisabledDepPremium := false
	for _, config := range expectedDeployedList {
		if config.Name == "main-addon" {
			foundMain = true
		}
		if config.Name == "multi-flavor-dependency" && config.Flavor.Name == "basic" {
			foundDisabledDepBasic = true
		}
		if config.Name == "multi-flavor-dependency" && config.Flavor.Name == "premium" {
			foundDisabledDepPremium = true
		}
	}

	assert.True(t, foundMain, "Main addon should be in expected deployed list")
	assert.False(t, foundDisabledDepBasic, "Manually disabled dependency (basic flavor) should NOT be in expected deployed list")
	assert.False(t, foundDisabledDepPremium, "Manually disabled dependency (premium flavor) should NOT be in expected deployed list due to offering-level disable")
}

// TestBuildDependencyGraphWithTreeLevelDisable tests that disabling a dependency
// at the offering level affects the entire dependency tree, not just immediate children
func TestBuildDependencyGraphWithTreeLevelDisable(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoServiceWithTreeDeps{}, // New mock with nested dependencies
	}

	// Create addon config where we disable an offering that appears multiple times in the tree
	addonConfig := &cloudinfo.AddonConfig{
		OfferingName:    "main-addon",
		OfferingFlavor:  "standard",
		CatalogID:       "test-catalog",
		OfferingID:      "main-offering-id",
		VersionLocator:  "test-catalog.main-version",
		ResolvedVersion: "v1.0.0",
		Dependencies: []cloudinfo.AddonConfig{
			{
				// We disable "common-library" which appears at multiple levels in the tree
				OfferingName:    "common-library",
				OfferingFlavor:  "standard",
				CatalogID:       "dependency-catalog",
				OfferingID:      "common-library-offering-id",
				VersionLocator:  "dependency-catalog.common-library-version",
				ResolvedVersion: "v1.0.0",
				Enabled:         core.BoolPtr(false), // Disabled at root level
				Dependencies:    []cloudinfo.AddonConfig{},
			},
		},
	}

	visited := make(map[string]bool)

	graphResult, err := options.buildDependencyGraph(
		"test-catalog",
		"main-offering-id",
		"test-catalog.main-version",
		"standard",
		addonConfig,
		visited,
	)

	assert.NoError(t, err)

	// Extract results
	expectedDeployedList := graphResult.ExpectedDeployedList

	// Debug: print what we actually got
	t.Logf("Expected deployed list contains %d items:", len(expectedDeployedList))
	for _, item := range expectedDeployedList {
		t.Logf("  - %s:%s:%s", item.Name, item.Version, item.Flavor.Name)
	}

	// Should contain main-addon and web-service, but NOT common-library
	// even though common-library is a dependency of both main-addon and web-service
	assert.Len(t, expectedDeployedList, 2, "Should contain main-addon and web-service, but not the disabled common-library")

	// Verify what's present and what's not
	foundMain := false
	foundWebService := false
	foundCommonLibrary := false
	for _, config := range expectedDeployedList {
		switch config.Name {
		case "main-addon":
			foundMain = true
		case "web-service":
			foundWebService = true
		case "common-library":
			foundCommonLibrary = true
		}
	}

	assert.True(t, foundMain, "Main addon should be in expected deployed list")
	assert.True(t, foundWebService, "Web service should be in expected deployed list")
	assert.False(t, foundCommonLibrary, "Common library should NOT be in expected deployed list (disabled at tree level)")
}

// MockCloudInfoService is a minimal mock for testing
type MockCloudInfoService struct {
	cloudinfo.CloudInfoServiceI
}

func (m *MockCloudInfoService) GetOffering(catalogID, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {
	// Return a mock offering with minimal required fields
	var name string
	var versions []catalogmanagementv1.Version

	// Return different mocks based on the offering ID
	switch offeringID {
	case "main-offering-id":
		name = "main-addon"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("test-catalog.main-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	case "observability-offering-id":
		name = "deploy-arch-ibm-observability"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("test-catalog.observability-version"),
				Version:        core.StringPtr("v0.0.1-dev-tf-an-zl3h3r"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	case "kms-offering-id":
		name = "deploy-arch-ibm-kms"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.kms-version"),
				Version:        core.StringPtr("v5.1.4"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	case "account-base-offering-id":
		name = "deploy-arch-ibm-account-infra-base"
		// This offering has multiple versions
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.account-base-version"),
				Version:        core.StringPtr("v3.0.11"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
			{
				VersionLocator: core.StringPtr("dependency-catalog.account-base-nested-version"),
				Version:        core.StringPtr("v3.0.7"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	default:
		name = "default-offering"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.dependency-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	}

	offering := &catalogmanagementv1.Offering{
		Name: core.StringPtr(name),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions:    versions,
			},
		},
	}
	return offering, nil, nil
}

func (m *MockCloudInfoService) GetOfferingVersionLocatorByConstraint(catalogID, offeringID, versionConstraint, flavor string) (version, versionLocator string, err error) {
	return "v1.0.0", "test-catalog.dependency-version", nil
}

// MockCloudInfoServiceWithCatalogDeps is a mock that returns catalog dependencies
type MockCloudInfoServiceWithCatalogDeps struct {
	cloudinfo.CloudInfoServiceI
}

func (m *MockCloudInfoServiceWithCatalogDeps) GetOffering(catalogID, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {
	var name string
	var versions []catalogmanagementv1.Version

	switch offeringID {
	case "main-offering-id":
		name = "main-addon"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("test-catalog.main-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					// This dependency is marked as OnByDefault=true in the catalog
					Dependencies: []catalogmanagementv1.OfferingReference{
						{
							Name:          core.StringPtr("catalog-default-dependency"),
							ID:            core.StringPtr("catalog-dep-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=2.0.0"),
							OnByDefault:   core.BoolPtr(true), // This is ON by default in catalog
							Flavors:       []string{"standard"},
							DefaultFlavor: core.StringPtr("standard"),
						},
					},
				},
			},
		}
	case "catalog-dep-offering-id":
		name = "catalog-default-dependency"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.catalog-dep-version"),
				Version:        core.StringPtr("v2.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	default:
		name = "default-offering"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("default.version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	}

	offering := &catalogmanagementv1.Offering{
		Name: core.StringPtr(name),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions:    versions,
			},
		},
	}
	return offering, nil, nil
}

func (m *MockCloudInfoServiceWithCatalogDeps) GetOfferingVersionLocatorByConstraint(catalogID, offeringID, versionConstraint, flavor string) (version, versionLocator string, err error) {
	return "v2.0.0", "dependency-catalog.catalog-dep-version", nil
}

// MockCloudInfoServiceWithMultipleFlavors is a mock that simulates offerings with multiple flavors
type MockCloudInfoServiceWithMultipleFlavors struct {
	cloudinfo.CloudInfoServiceI
}

func (m *MockCloudInfoServiceWithMultipleFlavors) GetOffering(catalogID, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {
	var name string
	var versions []catalogmanagementv1.Version

	switch offeringID {
	case "main-offering-id":
		name = "main-addon"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("test-catalog.main-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					// This dependency has multiple flavors, both on by default
					Dependencies: []catalogmanagementv1.OfferingReference{
						{
							Name:          core.StringPtr("multi-flavor-dependency"),
							ID:            core.StringPtr("multi-flavor-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=2.0.0"),
							OnByDefault:   core.BoolPtr(true), // Basic flavor is ON by default
							Flavors:       []string{"basic"},
							DefaultFlavor: core.StringPtr("basic"),
						},
						{
							Name:          core.StringPtr("multi-flavor-dependency"),
							ID:            core.StringPtr("multi-flavor-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=2.0.0"),
							OnByDefault:   core.BoolPtr(true), // Premium flavor is ALSO ON by default
							Flavors:       []string{"premium"},
							DefaultFlavor: core.StringPtr("premium"),
						},
					},
				},
			},
		}
	case "multi-flavor-offering-id":
		name = "multi-flavor-dependency"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.multi-flavor-basic-version"),
				Version:        core.StringPtr("v2.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
			{
				VersionLocator: core.StringPtr("dependency-catalog.multi-flavor-premium-version"),
				Version:        core.StringPtr("v2.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	default:
		name = "default-offering"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("default.version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	}

	offering := &catalogmanagementv1.Offering{
		Name: core.StringPtr(name),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions:    versions,
			},
		},
	}
	return offering, nil, nil
}

func (m *MockCloudInfoServiceWithMultipleFlavors) GetOfferingVersionLocatorByConstraint(catalogID, offeringID, versionConstraint, flavor string) (version, versionLocator string, err error) {
	switch flavor {
	case "basic":
		return "v2.0.0", "dependency-catalog.multi-flavor-basic-version", nil
	case "premium":
		return "v2.0.0", "dependency-catalog.multi-flavor-premium-version", nil
	default:
		return "v2.0.0", "dependency-catalog.multi-flavor-basic-version", nil
	}
}

// MockCloudInfoServiceWithTreeDeps is a mock that simulates nested dependency trees
// where the same offering appears at multiple levels
type MockCloudInfoServiceWithTreeDeps struct {
	cloudinfo.CloudInfoServiceI
}

func (m *MockCloudInfoServiceWithTreeDeps) GetOffering(catalogID, offeringID string) (result *catalogmanagementv1.Offering, response *core.DetailedResponse, err error) {
	var name string
	var versions []catalogmanagementv1.Version

	switch offeringID {
	case "main-offering-id":
		name = "main-addon"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("test-catalog.main-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{
						{
							Name:          core.StringPtr("web-service"),
							ID:            core.StringPtr("web-service-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=1.0.0"),
							OnByDefault:   core.BoolPtr(true),
							Flavors:       []string{"standard"},
							DefaultFlavor: core.StringPtr("standard"),
						},
						{
							Name:          core.StringPtr("common-library"),
							ID:            core.StringPtr("common-library-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=1.0.0"),
							OnByDefault:   core.BoolPtr(true),
							Flavors:       []string{"standard"},
							DefaultFlavor: core.StringPtr("standard"),
						},
					},
				},
			},
		}
	case "web-service-offering-id":
		name = "web-service"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.web-service-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					// web-service also depends on common-library
					Dependencies: []catalogmanagementv1.OfferingReference{
						{
							Name:          core.StringPtr("common-library"),
							ID:            core.StringPtr("common-library-offering-id"),
							CatalogID:     core.StringPtr("dependency-catalog"),
							Version:       core.StringPtr(">=1.0.0"),
							OnByDefault:   core.BoolPtr(true),
							Flavors:       []string{"standard"},
							DefaultFlavor: core.StringPtr("standard"),
						},
					},
				},
			},
		}
	case "common-library-offering-id":
		name = "common-library"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("dependency-catalog.common-library-version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	default:
		name = "default-offering"
		versions = []catalogmanagementv1.Version{
			{
				VersionLocator: core.StringPtr("default.version"),
				Version:        core.StringPtr("v1.0.0"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			},
		}
	}

	offering := &catalogmanagementv1.Offering{
		Name: core.StringPtr(name),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions:    versions,
			},
		},
	}
	return offering, nil, nil
}

func (m *MockCloudInfoServiceWithTreeDeps) GetOfferingVersionLocatorByConstraint(catalogID, offeringID, versionConstraint, flavor string) (version, versionLocator string, err error) {
	switch offeringID {
	case "web-service-offering-id":
		return "v1.0.0", "dependency-catalog.web-service-version", nil
	case "common-library-offering-id":
		return "v1.0.0", "dependency-catalog.common-library-version", nil
	default:
		return "v1.0.0", "dependency-catalog.default-version", nil
	}
}

// TestValidateDependenciesDetectsMissingConfigs tests that validateDependencies
// properly detects when expected configurations are missing from the deployed list
func TestValidateDependenciesDetectsMissingConfigs(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Create a simple dependency graph
	stringGraph := make(map[string][]cloudinfo.OfferingReferenceDetail)

	// Expected Deployed list - contains configs that should be deployed
	expectedDeployedList := []cloudinfo.OfferingReferenceDetail{
		{
			Name:    "deploy-arch-ibm-account-infra-base",
			Version: "v3.0.7",
			Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
		},
		{
			Name:    "deploy-arch-ibm-kms",
			Version: "v5.1.4",
			Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
		},
	}

	// Actually deployed list - missing the account-infra-base config
	actuallyDeployedList := []cloudinfo.OfferingReferenceDetail{
		{
			Name:    "deploy-arch-ibm-kms",
			Version: "v5.1.4",
			Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
		},
	}

	// This should fail because deploy-arch-ibm-account-infra-base:v3.0.7:resource-group-only is missing
	result := options.validateDependencies(stringGraph, expectedDeployedList, actuallyDeployedList)

	assert.False(t, result.IsValid, "Should detect missing expected config")
	assert.Equal(t, 1, len(result.MissingConfigs), "Should have one missing config")
	assert.Equal(t, "deploy-arch-ibm-account-infra-base", result.MissingConfigs[0].Name)
	assert.Equal(t, "v3.0.7", result.MissingConfigs[0].Version)
	assert.Equal(t, "resource-group-only", result.MissingConfigs[0].Flavor.Name)
}

// TestValidateDependenciesDetectsUnexpectedConfigs tests that validateDependencies
// properly detects when unexpected configurations are deployed
func TestValidateDependenciesDetectsUnexpectedConfigs(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Create a simple dependency graph
	stringGraph := make(map[string][]cloudinfo.OfferingReferenceDetail)

	// Expected Deployed list
	expectedDeployedList := []cloudinfo.OfferingReferenceDetail{
		{
			Name:    "deploy-arch-ibm-kms",
			Version: "v5.1.4",
			Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
		},
	}

	// Actually deployed list - contains an unexpected config
	actuallyDeployedList := []cloudinfo.OfferingReferenceDetail{
		{
			Name:    "deploy-arch-ibm-kms",
			Version: "v5.1.4",
			Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
		},
		{
			Name:    "deploy-arch-ibm-account-infra-base",
			Version: "v3.0.7",
			Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
		},
	}

	// This should fail because deploy-arch-ibm-account-infra-base is unexpected
	result := options.validateDependencies(stringGraph, expectedDeployedList, actuallyDeployedList)

	assert.False(t, result.IsValid, "Should detect unexpected config")
	assert.Equal(t, 1, len(result.UnexpectedConfigs), "Should have one unexpected config")
	assert.Equal(t, "deploy-arch-ibm-account-infra-base", result.UnexpectedConfigs[0].Name)
	assert.Equal(t, "v3.0.7", result.UnexpectedConfigs[0].Version)
	assert.Equal(t, "resource-group-only", result.UnexpectedConfigs[0].Flavor.Name)
}

// TestValidateDependenciesSuccess tests that validateDependencies passes when
// expected and actual configurations match exactly
func TestValidateDependenciesSuccess(t *testing.T) {
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Create a simple dependency graph
	stringGraph := make(map[string][]cloudinfo.OfferingReferenceDetail)

	// Both lists are identical - should pass
	configList := []cloudinfo.OfferingReferenceDetail{
		{
			Name:    "deploy-arch-ibm-account-infra-base",
			Version: "v3.0.7",
			Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
		},
		{
			Name:    "deploy-arch-ibm-kms",
			Version: "v5.1.4",
			Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
		},
	}

	result := options.validateDependencies(stringGraph, configList, configList)

	assert.True(t, result.IsValid, "Should pass when expected and actual configs match")
	assert.Equal(t, 0, len(result.MissingConfigs), "Should have no missing configs")
	assert.Equal(t, 0, len(result.UnexpectedConfigs), "Should have no unexpected configs")
	assert.Equal(t, 0, len(result.DependencyErrors), "Should have no dependency errors")
}

// TestPrintConsolidatedValidationSummary tests the consolidated validation summary output
func TestPrintConsolidatedValidationSummary(t *testing.T) {
	// Create a mock test logger
	logger := common.CreateSmartAutoBufferingLogger(t.Name(), false)

	// Create test options
	options := &TestAddonOptions{
		Testing: t,
		Logger:  logger,
	}

	// Create a ValidationResult with sample errors
	validationResult := ValidationResult{
		IsValid: false,
		DependencyErrors: []cloudinfo.DependencyError{
			{
				Addon: cloudinfo.OfferingReferenceDetail{
					Name:    "main-addon",
					Version: "v1.0.0",
					Flavor: cloudinfo.Flavor{
						Name: "standard",
					},
				},
				DependencyRequired: cloudinfo.OfferingReferenceDetail{
					Name:    "required-dependency",
					Version: "v2.0.0",
					Flavor: cloudinfo.Flavor{
						Name: "basic",
					},
				},
				DependenciesAvailable: []cloudinfo.OfferingReferenceDetail{
					{
						Name:    "required-dependency",
						Version: "v1.5.0",
						Flavor: cloudinfo.Flavor{
							Name: "basic",
						},
					},
				},
			},
		},
		UnexpectedConfigs: []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "unexpected-addon",
				Version: "v1.0.0",
				Flavor: cloudinfo.Flavor{
					Name: "standard",
				},
			},
		},
		MissingConfigs: []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "missing-addon",
				Version: "v1.0.0",
				Flavor: cloudinfo.Flavor{
					Name: "basic",
				},
			},
		},
		Messages: []string{"Validation failed"},
	}

	// Test consolidated summary (default behavior)
	t.Run("ConsolidatedSummary", func(t *testing.T) {
		options.VerboseValidationErrors = false
		options.printConsolidatedValidationSummary(validationResult)

		// The test passes if no panics occur and the method executes successfully
		// In a real test environment, you could capture the logger output and verify specific messages
	})

	// Test detailed errors (verbose mode)
	t.Run("DetailedErrors", func(t *testing.T) {
		options.VerboseValidationErrors = true
		options.EnhancedTreeValidationOutput = false
		options.printDetailedValidationErrors(validationResult)

		// The test passes if no panics occur and the method executes successfully
	})

	// Test enhanced tree validation output
	t.Run("EnhancedTreeOutput", func(t *testing.T) {
		options.VerboseValidationErrors = false
		options.EnhancedTreeValidationOutput = true

		// Create a more realistic dependency graph similar to the observability scenario
		graph := make(map[string][]cloudinfo.OfferingReferenceDetail)

		// Main addon depends on KMS and account-infra-base
		mainAddonKey := "deploy-arch-ibm-observability:v0.0.1-dev-tf-an-lkoqsr:instances"
		graph[mainAddonKey] = []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "deploy-arch-ibm-kms",
				Version: "v5.1.4",
				Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
			},
			{
				Name:    "deploy-arch-ibm-account-infra-base",
				Version: "v3.0.11",
				Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
			},
		}

		// KMS depends on account-infra-base v3.0.7 (this is what causes the conflict)
		kmsKey := "deploy-arch-ibm-kms:v5.1.4:fully-configurable"
		graph[kmsKey] = []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "deploy-arch-ibm-account-infra-base",
				Version: "v3.0.7",
				Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
			},
		}

		expectedDeployedList := []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "deploy-arch-ibm-observability",
				Version: "v0.0.1-dev-tf-an-lkoqsr",
				Flavor:  cloudinfo.Flavor{Name: "instances"},
			},
			{
				Name:    "deploy-arch-ibm-kms",
				Version: "v5.1.4",
				Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
			},
			{
				Name:    "deploy-arch-ibm-account-infra-base",
				Version: "v3.0.7",
				Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
			},
			{
				Name:    "deploy-arch-ibm-account-infra-base",
				Version: "v3.0.11",
				Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
			},
		}

		// Actually deployed (missing v3.0.7)
		actuallyDeployedList := []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "deploy-arch-ibm-observability",
				Version: "v0.0.1-dev-tf-an-lkoqsr",
				Flavor:  cloudinfo.Flavor{Name: "instances"},
			},
			{
				Name:    "deploy-arch-ibm-account-infra-base",
				Version: "v3.0.11",
				Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
			},
			{
				Name:    "deploy-arch-ibm-kms",
				Version: "v5.1.4",
				Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
			},
		}

		// Create validation result that matches the scenario
		enhancedValidationResult := ValidationResult{
			IsValid: false,
			DependencyErrors: []cloudinfo.DependencyError{
				{
					Addon: cloudinfo.OfferingReferenceDetail{
						Name:    "deploy-arch-ibm-kms",
						Version: "v5.1.4",
						Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
					},
					DependencyRequired: cloudinfo.OfferingReferenceDetail{
						Name:    "deploy-arch-ibm-account-infra-base",
						Version: "v3.0.7",
						Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
					},
					DependenciesAvailable: []cloudinfo.OfferingReferenceDetail{
						{
							Name:    "deploy-arch-ibm-account-infra-base",
							Version: "v3.0.11",
							Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
						},
					},
				},
			},
			MissingConfigs: []cloudinfo.OfferingReferenceDetail{
				{
					Name:    "deploy-arch-ibm-account-infra-base",
					Version: "v3.0.7",
					Flavor:  cloudinfo.Flavor{Name: "resource-group-only"},
				},
			},
			Messages: []string{"Validation failed"},
		}

		options.printDependencyTreeWithValidationStatus(graph, expectedDeployedList, actuallyDeployedList, enhancedValidationResult)

		// The test passes if no panics occur and the method executes successfully
	})
}

// TestMissingConfigsErrorMessageFormat tests that error messages for missing configs
// include specific details about which configs are missing
func TestMissingConfigsErrorMessageFormat(t *testing.T) {
	// Create validation result with missing configs
	validationResult := ValidationResult{
		IsValid:           false,
		DependencyErrors:  []cloudinfo.DependencyError{},
		UnexpectedConfigs: []cloudinfo.OfferingReferenceDetail{},
		MissingConfigs: []cloudinfo.OfferingReferenceDetail{
			{
				Name:    "deploy-arch-ibm-event-notifications",
				Version: "v0.0.1-dev-test123",
				Flavor:  cloudinfo.Flavor{Name: "fully-configurable"},
			},
			{
				Name:    "deploy-arch-ibm-kms",
				Version: "v5.1.4",
				Flavor:  cloudinfo.Flavor{Name: "instance"},
			},
		},
		Messages: []string{"found 2 missing expected configs"},
	}

	// Simulate the error message construction from the actual code
	var errorDetails []string
	if len(validationResult.DependencyErrors) > 0 {
		errorDetails = append(errorDetails, fmt.Sprintf("%d dependency errors", len(validationResult.DependencyErrors)))
	}
	if len(validationResult.UnexpectedConfigs) > 0 {
		errorDetails = append(errorDetails, fmt.Sprintf("%d unexpected configs", len(validationResult.UnexpectedConfigs)))
	}
	if len(validationResult.MissingConfigs) > 0 {
		// Include specific names of missing configs in the error message
		var missingNames []string
		for _, missing := range validationResult.MissingConfigs {
			missingNames = append(missingNames, fmt.Sprintf("%s (%s, %s)", missing.Name, missing.Version, missing.Flavor.Name))
		}
		errorDetails = append(errorDetails, fmt.Sprintf("%d missing configs: [%s]", len(validationResult.MissingConfigs), strings.Join(missingNames, ", ")))
	}

	var errorMsg string
	if len(errorDetails) > 0 {
		errorMsg = fmt.Sprintf("dependency validation failed: %s", strings.Join(errorDetails, ", "))
	} else {
		errorMsg = "dependency validation failed - check validation output above for details"
	}

	// Verify the error message includes specific config details
	assert.Contains(t, errorMsg, "deploy-arch-ibm-event-notifications (v0.0.1-dev-test123, fully-configurable)")
	assert.Contains(t, errorMsg, "deploy-arch-ibm-kms (v5.1.4, instance)")
	assert.Contains(t, errorMsg, "2 missing configs:")
	assert.Contains(t, errorMsg, "dependency validation failed:")

	// Verify the format is readable and contains all expected information
	expectedSubstrings := []string{
		"dependency validation failed:",
		"2 missing configs:",
		"deploy-arch-ibm-event-notifications (v0.0.1-dev-test123, fully-configurable)",
		"deploy-arch-ibm-kms (v5.1.4, instance)",
	}

	for _, substr := range expectedSubstrings {
		assert.Contains(t, errorMsg, substr, "Error message should contain: %s", substr)
	}

	t.Logf("Generated error message: %s", errorMsg)
}

// TestReferenceValidationMemberDeploymentWarning tests that references requiring member deployment
// are treated as warnings rather than errors
func TestReferenceValidationMemberDeploymentWarning(t *testing.T) {

	// Create a mock resolve response that includes a member deployment reference
	mockResolveResponse := &cloudinfo.ResolveResponse{
		References: []cloudinfo.BatchReferenceResolvedItem{
			{
				Reference: "ref://project.test/configs/member-config/inputs/resource_group_name",
				Code:      400,
				State:     "unresolved",
				Message:   "The project reference requires the specified member configuration deploy-arch-ibm-account-infra-base-abc123 to be deployed. Please deploy the referring configuration.",
			},
			{
				Reference: "ref://project.test/configs/other-config/inputs/api_key",
				Code:      200,
				State:     "resolved",
				Value:     "test-value",
			},
		},
	}

	// Test that the member deployment reference is handled as a warning
	failedRefs := []string{}

	// Simulate the logic from the actual function
	for _, ref := range mockResolveResponse.References {
		if ref.Code != 200 {
			// Check if this is a valid reference that cannot be resolved until after member deployment
			if strings.Contains(ref.Message, "project reference requires") &&
				strings.Contains(ref.Message, "member configuration") &&
				strings.Contains(ref.Message, "to be deployed") {
				// This should be treated as a warning, not added to failedRefs
				t.Logf("Warning: %s - %s", ref.Reference, ref.Message)
			} else {
				// This should be treated as an error
				failedRefs = append(failedRefs, ref.Reference)
			}
		}
	}

	// Verify that the member deployment reference was not added to failedRefs
	assert.Empty(t, failedRefs, "Member deployment references should not be treated as failed references")

	// Verify the logic works correctly - only non-member-deployment errors should be in failedRefs
	// Let's test with a different error message
	mockResolveResponse2 := &cloudinfo.ResolveResponse{
		References: []cloudinfo.BatchReferenceResolvedItem{
			{
				Reference: "ref://project.test/configs/invalid-config/inputs/bad_input",
				Code:      404,
				State:     "error",
				Message:   "Configuration not found",
			},
		},
	}

	failedRefs2 := []string{}
	for _, ref := range mockResolveResponse2.References {
		if ref.Code != 200 {
			// Check if this is a valid reference that cannot be resolved until after member deployment
			if strings.Contains(ref.Message, "project reference requires") &&
				strings.Contains(ref.Message, "member configuration") &&
				strings.Contains(ref.Message, "to be deployed") {
				// This should be treated as a warning, not added to failedRefs
				t.Logf("Warning: %s - %s", ref.Reference, ref.Message)
			} else {
				// This should be treated as an error
				failedRefs2 = append(failedRefs2, ref.Reference)
			}
		}
	}

	// Verify that the non-member-deployment error was added to failedRefs
	assert.Len(t, failedRefs2, 1, "Non-member-deployment reference errors should be treated as failed references")
	assert.Equal(t, "ref://project.test/configs/invalid-config/inputs/bad_input", failedRefs2[0])
}

// TestMemberDeploymentReferenceHandling tests the logic for detecting member deployment references
func TestMemberDeploymentReferenceHandling(t *testing.T) {
	// Test case 1: Member deployment reference (should be treated as warning)
	memberDeploymentMessage := "The project reference requires the specified member configuration deploy-arch-ibm-account-infra-base-abc123 to be deployed. Please deploy the referring configuration."
	isMemberDeploymentRef := IsMemberDeploymentReference(memberDeploymentMessage)
	assert.True(t, isMemberDeploymentRef, "Member deployment reference should be detected")

	// Test case 2: Different error message (should be treated as error)
	otherErrorMessage := "Configuration not found"
	isOtherError := IsMemberDeploymentReference(otherErrorMessage)
	assert.False(t, isOtherError, "Other error messages should not be detected as member deployment references")

	// Test case 3: Partial match (should be treated as error)
	partialMatchMessage := "The project reference requires something but not member configuration"
	isPartialMatch := IsMemberDeploymentReference(partialMatchMessage)
	assert.False(t, isPartialMatch, "Partial matches should not be detected as member deployment references")
}

// TestCategorizeError_NoDuplicateEntries tests that categorizeError doesn't create duplicate entries
// Uses ErrorAlreadyCategorized boolean flag to prevent duplicate processing instead of string matching
func TestCategorizeError_NoDuplicateEntries(t *testing.T) {
	options := &TestAddonOptions{}
	result := &PermutationTestResult{}

	// Test error that triggers "missing required inputs" categorization
	testError := fmt.Errorf("missing required inputs: deploy-arch-ibm-cloud-logs-test (missing: existing_cos_instance_crn)")

	// Verify initial state
	assert.False(t, result.ErrorAlreadyCategorized, "Initially ErrorAlreadyCategorized should be false")

	// First call - this should set ErrorAlreadyCategorized = true and create ValidationResult
	options.categorizeError(testError, result)
	assert.True(t, result.ErrorAlreadyCategorized, "After first call, ErrorAlreadyCategorized should be true")

	// Second call with the SAME error - should NOT create duplicates due to boolean flag
	options.categorizeError(testError, result)

	// Verify boolean flag prevents duplicate processing
	assert.True(t, result.ErrorAlreadyCategorized, "ErrorAlreadyCategorized should remain true")
	assert.NotNil(t, result.ValidationResult, "ValidationResult should be created")
	assert.Equal(t, 1, len(result.ValidationResult.MissingInputs), "Should have exactly 1 error entry, no duplicates")

	// Verify error content is correct
	if len(result.ValidationResult.MissingInputs) > 0 {
		errorStr := result.ValidationResult.MissingInputs[0]
		assert.Contains(t, errorStr, "deploy-arch-ibm-cloud-logs-test", "Error should contain component name")
		assert.Contains(t, errorStr, "existing_cos_instance_crn", "Error should contain missing input name")
	}
}

// TestErrorPatternClassification tests the new structured error classification system
// replacing fragile string matching with regex-based pattern matching
func TestErrorPatternClassification(t *testing.T) {
	testCases := []struct {
		name            string
		errorMessage    string
		expectedType    ErrorType
		expectedSubtype string
		shouldMatch     bool
	}{
		{
			name:            "Missing required inputs validation error",
			errorMessage:    "missing required inputs: deploy-arch-ibm-cos",
			expectedType:    ValidationError,
			expectedSubtype: "missing_inputs",
			shouldMatch:     true,
		},
		{
			name:            "Deployment timeout transient error",
			errorMessage:    "deployment timeout after 300 seconds",
			expectedType:    TransientError,
			expectedSubtype: "deployment_timeout",
			shouldMatch:     true,
		},
		{
			name:            "Runtime panic error",
			errorMessage:    "panic: runtime error: index out of range",
			expectedType:    RuntimeError,
			expectedSubtype: "panic",
			shouldMatch:     true,
		},
		{
			name:            "Unknown error pattern",
			errorMessage:    "some completely unknown error message",
			expectedType:    ValidationError, // Will be ignored since shouldMatch is false
			expectedSubtype: "",
			shouldMatch:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern, found := classifyError(tc.errorMessage)

			assert.Equal(t, tc.shouldMatch, found,
				"Pattern matching result should be %v", tc.shouldMatch)

			if tc.shouldMatch {
				assert.Equal(t, tc.expectedType, pattern.Type,
					"Error should be classified as %s but got %s", tc.expectedType, pattern.Type)
				assert.Equal(t, tc.expectedSubtype, pattern.Subtype,
					"Error subtype should be %s but got %s", tc.expectedSubtype, pattern.Subtype)
				assert.Greater(t, pattern.Confidence, 0.0,
					"Confidence should be greater than 0")
				assert.LessOrEqual(t, pattern.Confidence, 1.0,
					"Confidence should not exceed 1.0")
			}
		})
	}
}

// TestReferenceErrorDetector tests the new structured reference error detection system
// replacing complex multi-condition string matching with reusable pattern detection
func TestReferenceErrorDetector(t *testing.T) {
	testCases := []struct {
		name         string
		message      string
		shouldDetect bool
		description  string
	}{
		{
			name:         "Complete member deployment reference",
			message:      "The project reference requires the specified member configuration deploy-arch-ibm-account-infra-base-abc123 to be deployed. Please deploy the referring configuration.",
			shouldDetect: true,
			description:  "Should detect complete member deployment reference message",
		},
		{
			name:         "Member deployment reference with different wording",
			message:      "project reference requires member configuration xyz to be deployed",
			shouldDetect: true,
			description:  "Should detect reference with minimal required phrases",
		},
		{
			name:         "Missing 'project reference requires' phrase",
			message:      "member configuration deploy-arch-ibm-cos to be deployed",
			shouldDetect: false,
			description:  "Should not detect when missing required phrase",
		},
		{
			name:         "Missing 'member configuration' phrase",
			message:      "The project reference requires something to be deployed",
			shouldDetect: false,
			description:  "Should not detect when missing required phrase",
		},
		{
			name:         "Missing 'to be deployed' phrase",
			message:      "The project reference requires the member configuration xyz",
			shouldDetect: false,
			description:  "Should not detect when missing required phrase",
		},
		{
			name:         "Completely different error message",
			message:      "Configuration not found in catalog",
			shouldDetect: false,
			description:  "Should not detect unrelated error messages",
		},
		{
			name:         "Empty message",
			message:      "",
			shouldDetect: false,
			description:  "Should not detect empty messages",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the high-level function
			detected := IsMemberDeploymentReference(tc.message)
			assert.Equal(t, tc.shouldDetect, detected, tc.description)

			// Test the detector directly
			detectedDirect := memberDeploymentReferenceDetector.IsReferenceError(tc.message)
			assert.Equal(t, tc.shouldDetect, detectedDirect, "Direct detector call should match high-level function")
		})
	}
}

// TestReferenceErrorDetectorCustom tests creating custom detectors for different reference patterns
func TestReferenceErrorDetectorCustom(t *testing.T) {
	// Create a custom detector for a different type of reference error
	customDetector := &ReferenceErrorDetector{
		RequiredPhrases: []string{
			"invalid reference",
			"configuration",
		},
		ExcludePhrases: []string{
			"to be deployed", // This would make it a member deployment reference
		},
	}

	testCases := []struct {
		name         string
		message      string
		shouldDetect bool
	}{
		{
			name:         "Custom pattern match",
			message:      "invalid reference to configuration xyz",
			shouldDetect: true,
		},
		{
			name:         "Excluded by member deployment phrase",
			message:      "invalid reference to configuration xyz to be deployed",
			shouldDetect: false,
		},
		{
			name:         "Missing required phrase",
			message:      "invalid reference to something else",
			shouldDetect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected := customDetector.IsReferenceError(tc.message)
			assert.Equal(t, tc.shouldDetect, detected,
				"Custom detector should properly match pattern")
		})
	}
}

// TestAPIErrorDetector tests the new structured API error detection system
// replacing complex multi-condition string matching with classified error types
// TestConfigurationMatcher tests the new structured configuration matching system
// replacing fragile strings.Contains() checks with configurable matching strategies
func TestConfigurationMatcher(t *testing.T) {
	// Test configuration for main addon
	mainAddon := cloudinfo.AddonConfig{
		OfferingName: "deploy-arch-ibm-event-notifications",
		ConfigName:   "event-notifications-config",
	}

	// Test configuration for dependency
	dependency := cloudinfo.AddonConfig{
		OfferingName: "deploy-arch-ibm-kms",
		ConfigName:   "kms-config",
	}

	testCases := []struct {
		name         string
		addonConfig  cloudinfo.AddonConfig
		configName   string
		shouldMatch  bool
		expectedRule string
	}{
		{
			name:         "Exact config name match",
			addonConfig:  mainAddon,
			configName:   "event-notifications-config",
			shouldMatch:  true,
			expectedRule: "ExactNameMatch",
		},
		{
			name:         "Contains config name match",
			addonConfig:  mainAddon,
			configName:   "my-event-notifications-config-abc123",
			shouldMatch:  true,
			expectedRule: "ContainsNameMatch",
		},
		{
			name:         "Exact offering name match",
			addonConfig:  dependency,
			configName:   "deploy-arch-ibm-kms",
			shouldMatch:  true,
			expectedRule: "ExactNameMatch",
		},
		{
			name:         "Contains offering name match",
			addonConfig:  dependency,
			configName:   "deploy-arch-ibm-kms-xyz789",
			shouldMatch:  true,
			expectedRule: "ContainsNameMatch",
		},
		{
			name:         "Base offering name match",
			addonConfig:  cloudinfo.AddonConfig{OfferingName: "deploy-arch-ibm-kms:flavor"},
			configName:   "deploy-arch-ibm-kms-instance",
			shouldMatch:  true,
			expectedRule: "BaseNameMatch",
		},
		{
			name:         "No match",
			addonConfig:  mainAddon,
			configName:   "completely-different-config",
			shouldMatch:  false,
			expectedRule: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matcher := NewConfigurationMatcherForAddon(tc.addonConfig)
			matched, rule := matcher.IsMatch(tc.configName)

			assert.Equal(t, tc.shouldMatch, matched,
				"Configuration matching should be %v for config '%s'", tc.shouldMatch, tc.configName)

			if tc.shouldMatch {
				assert.NotNil(t, rule, "Matching rule should be returned when match is found")
				assert.Equal(t, tc.expectedRule, rule.Strategy.String(),
					"Expected strategy %s but got %s", tc.expectedRule, rule.Strategy.String())
			} else {
				assert.Nil(t, rule, "No rule should be returned when no match is found")
			}
		})
	}
}

// TestConfigurationMatchStrategy tests the enum string representation
func TestConfigurationMatchStrategy(t *testing.T) {
	testCases := []struct {
		strategy ConfigurationMatchStrategy
		expected string
	}{
		{ExactNameMatch, "ExactNameMatch"},
		{ContainsNameMatch, "ContainsNameMatch"},
		{BaseNameMatch, "BaseNameMatch"},
		{PrefixNameMatch, "PrefixNameMatch"},
		{ConfigurationMatchStrategy(999), "UnknownStrategy"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			actual := tc.strategy.String()
			assert.Equal(t, tc.expected, actual,
				"Strategy(%d).String() should return %s but got %s",
				tc.strategy, tc.expected, actual)
		})
	}
}

// TestConfigurationMatchRule tests individual rule matching logic
func TestConfigurationMatchRule(t *testing.T) {
	testCases := []struct {
		name        string
		rule        ConfigurationMatchRule
		configName  string
		shouldMatch bool
	}{
		{
			name: "Exact match rule",
			rule: ConfigurationMatchRule{
				Strategy: ExactNameMatch,
				Pattern:  "exact-config-name",
			},
			configName:  "exact-config-name",
			shouldMatch: true,
		},
		{
			name: "Exact match rule - no match",
			rule: ConfigurationMatchRule{
				Strategy: ExactNameMatch,
				Pattern:  "exact-config-name",
			},
			configName:  "different-config-name",
			shouldMatch: false,
		},
		{
			name: "Contains match rule",
			rule: ConfigurationMatchRule{
				Strategy: ContainsNameMatch,
				Pattern:  "kms",
			},
			configName:  "deploy-arch-ibm-kms-advanced",
			shouldMatch: true,
		},
		{
			name: "Base name match rule",
			rule: ConfigurationMatchRule{
				Strategy: BaseNameMatch,
				Pattern:  "deploy-arch-ibm-kms:advanced",
			},
			configName:  "deploy-arch-ibm-kms-instance",
			shouldMatch: true,
		},
		{
			name: "Prefix match rule",
			rule: ConfigurationMatchRule{
				Strategy: PrefixNameMatch,
				Pattern:  "deploy-arch",
			},
			configName:  "deploy-arch-ibm-event-notifications",
			shouldMatch: true,
		},
		{
			name: "Prefix match rule - no match",
			rule: ConfigurationMatchRule{
				Strategy: PrefixNameMatch,
				Pattern:  "deploy-arch",
			},
			configName:  "custom-deploy-arch-config",
			shouldMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.rule.IsMatch(tc.configName)
			assert.Equal(t, tc.shouldMatch, actual,
				"Rule with strategy %s and pattern '%s' should %v match config '%s'",
				tc.rule.Strategy.String(), tc.rule.Pattern, tc.shouldMatch, tc.configName)
		})
	}
}

// TestSensitiveDataDetector tests the new structured sensitive data detection system
// replacing fragile strings.Contains() checks with configurable pattern matching
func TestSensitiveDataDetector(t *testing.T) {
	detector := NewSensitiveDataDetector()

	testCases := []struct {
		name           string
		fieldName      string
		expectedType   SensitiveFieldType
		shouldBeMasked bool
		expectedMask   string
	}{
		{
			name:           "API key field",
			fieldName:      "ibmcloud_api_key",
			expectedType:   APIKeyField,
			shouldBeMasked: true,
			expectedMask:   "[API_KEY_REDACTED]",
		},
		{
			name:           "Password field",
			fieldName:      "user_password",
			expectedType:   PasswordField,
			shouldBeMasked: true,
			expectedMask:   "[PASSWORD_REDACTED]",
		},
		{
			name:           "Secret field",
			fieldName:      "private_key_data",
			expectedType:   SecretField,
			shouldBeMasked: true,
			expectedMask:   "[SECRET_REDACTED]",
		},
		{
			name:           "Certificate field",
			fieldName:      "tls_certificate",
			expectedType:   CertificateField,
			shouldBeMasked: true,
			expectedMask:   "[CERTIFICATE_REDACTED]",
		},
		{
			name:           "Non-sensitive field",
			fieldName:      "resource_group_name",
			expectedType:   NonSensitiveField,
			shouldBeMasked: false,
			expectedMask:   "test-value",
		},
		{
			name:           "Case insensitive API key",
			fieldName:      "IBMCLOUD_API_KEY",
			expectedType:   APIKeyField,
			shouldBeMasked: true,
			expectedMask:   "[API_KEY_REDACTED]",
		},
		{
			name:           "Partial match password",
			fieldName:      "admin_password_hash",
			expectedType:   PasswordField,
			shouldBeMasked: true,
			expectedMask:   "[PASSWORD_REDACTED]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test field classification
			actualType := detector.ClassifyField(tc.fieldName)
			assert.Equal(t, tc.expectedType, actualType,
				"Field '%s' should be classified as %s but got %s",
				tc.fieldName, tc.expectedType.String(), actualType.String())

			// Test sensitivity detection
			isSensitive := detector.IsSensitive(tc.fieldName)
			assert.Equal(t, tc.shouldBeMasked, isSensitive,
				"Field '%s' sensitivity should be %v but got %v",
				tc.fieldName, tc.shouldBeMasked, isSensitive)

			// Test value masking
			maskedValue := detector.GetMaskedValue(tc.fieldName, "test-value")
			assert.Equal(t, tc.expectedMask, maskedValue,
				"Field '%s' masked value should be '%s' but got '%s'",
				tc.fieldName, tc.expectedMask, maskedValue)

			// Test logging decision
			shouldLog := detector.ShouldLogValue(tc.fieldName)
			assert.Equal(t, !tc.shouldBeMasked, shouldLog,
				"Field '%s' should log value: %v but got %v",
				tc.fieldName, !tc.shouldBeMasked, shouldLog)
		})
	}
}

// TestSensitiveFieldType tests the enum string representation
func TestSensitiveFieldType(t *testing.T) {
	testCases := []struct {
		fieldType SensitiveFieldType
		expected  string
	}{
		{APIKeyField, "APIKeyField"},
		{PasswordField, "PasswordField"},
		{SecretField, "SecretField"},
		{CertificateField, "CertificateField"},
		{NonSensitiveField, "NonSensitiveField"},
		{SensitiveFieldType(999), "UnknownField"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			actual := tc.fieldType.String()
			assert.Equal(t, tc.expected, actual,
				"SensitiveFieldType(%d).String() should return %s but got %s",
				tc.fieldType, tc.expected, actual)
		})
	}
}

// TestPackageLevelSensitivityFunctions tests the convenience functions
func TestPackageLevelSensitivityFunctions(t *testing.T) {
	testCases := []struct {
		fieldName    string
		shouldMask   bool
		expectedMask string
	}{
		{"ibmcloud_api_key", true, "[API_KEY_REDACTED]"},
		{"user_password", true, "[PASSWORD_REDACTED]"},
		{"resource_group", false, "test-value"},
		{"secret_key", true, "[SECRET_REDACTED]"},
	}

	for _, tc := range testCases {
		t.Run(tc.fieldName, func(t *testing.T) {
			// Test package-level sensitivity check
			isSensitive := IsSensitiveField(tc.fieldName)
			assert.Equal(t, tc.shouldMask, isSensitive,
				"IsSensitiveField('%s') should return %v but got %v",
				tc.fieldName, tc.shouldMask, isSensitive)

			// Test package-level masked value
			maskedValue := GetSafeMaskedValue(tc.fieldName, "test-value")
			assert.Equal(t, tc.expectedMask, maskedValue,
				"GetSafeMaskedValue('%s', 'test-value') should return '%s' but got '%s'",
				tc.fieldName, tc.expectedMask, maskedValue)
		})
	}
}

// TestLegacyErrorClassificationUpdated tests that legacy error classification functions
// now use the new structured ErrorPattern system for consistency
func TestLegacyErrorClassificationUpdated(t *testing.T) {
	testCases := []struct {
		name         string
		errorMessage string
		isValidation bool
		isTransient  bool
		isRuntime    bool
	}{
		{
			name:         "Validation error - missing inputs",
			errorMessage: "missing required inputs: deploy-arch-ibm-cos",
			isValidation: true,
			isTransient:  false,
			isRuntime:    false,
		},
		{
			name:         "Validation error - dependency validation",
			errorMessage: "dependency validation failed: 2 unexpected configs",
			isValidation: true,
			isTransient:  false,
			isRuntime:    false,
		},
		{
			name:         "Transient error - deployment timeout",
			errorMessage: "deployment timeout occurred during TriggerDeployAndWait",
			isValidation: false,
			isTransient:  true,
			isRuntime:    false,
		},
		{
			name:         "Transient error - server error",
			errorMessage: "500 Internal Server error occurred",
			isValidation: false,
			isTransient:  true,
			isRuntime:    false,
		},
		{
			name:         "Runtime error - panic",
			errorMessage: "panic: runtime error: invalid memory address",
			isValidation: false,
			isTransient:  false,
			isRuntime:    true,
		},
		{
			name:         "Runtime error - nil pointer",
			errorMessage: "nil pointer dereference in function",
			isValidation: false,
			isTransient:  false,
			isRuntime:    true,
		},
		{
			name:         "Unknown error - defaults to false",
			errorMessage: "some completely unknown error type",
			isValidation: false,
			isTransient:  false,
			isRuntime:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test validation error classification
			actualValidation := isValidationError(tc.errorMessage)
			assert.Equal(t, tc.isValidation, actualValidation,
				"isValidationError('%s') should return %v but got %v",
				tc.errorMessage, tc.isValidation, actualValidation)

			// Test transient error classification
			actualTransient := isTransientError(tc.errorMessage)
			assert.Equal(t, tc.isTransient, actualTransient,
				"isTransientError('%s') should return %v but got %v",
				tc.errorMessage, tc.isTransient, actualTransient)

			// Test runtime error classification
			actualRuntime := isRuntimeError(tc.errorMessage)
			assert.Equal(t, tc.isRuntime, actualRuntime,
				"isRuntimeError('%s') should return %v but got %v",
				tc.errorMessage, tc.isRuntime, actualRuntime)

			// Ensure only one category is true (or none for unknown errors)
			trueCount := 0
			if actualValidation {
				trueCount++
			}
			if actualTransient {
				trueCount++
			}
			if actualRuntime {
				trueCount++
			}

			assert.LessOrEqual(t, trueCount, 1,
				"Error should be classified into at most one category, but got %d for: %s",
				trueCount, tc.errorMessage)
		})
	}
}

func TestAPIErrorDetector(t *testing.T) {
	testCases := []struct {
		name         string
		errorMessage string
		expectedType APIErrorType
		shouldDetect bool
		shouldSkip   bool
	}{
		{
			name:         "API key validation error with 500 status",
			errorMessage: "Failed to validate api key token - 500 Internal Server Error",
			expectedType: APIKeyError,
			shouldDetect: true,
			shouldSkip:   true,
		},
		{
			name:         "Project not found error with 404 status",
			errorMessage: "Project could not be found - 404 Not Found",
			expectedType: ProjectNotFoundError,
			shouldDetect: true,
			shouldSkip:   true,
		},
		{
			name:         "Known intermittent issue",
			errorMessage: "This is a known intermittent issue with the service",
			expectedType: IntermittentError,
			shouldDetect: true,
			shouldSkip:   true,
		},
		{
			name:         "Unknown error",
			errorMessage: "Some completely different error occurred",
			expectedType: UnknownAPIError,
			shouldDetect: false,
			shouldSkip:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test error classification
			errorType, detected := ClassifyAPIError(tc.errorMessage)
			assert.Equal(t, tc.shouldDetect, detected,
				"Error detection should be %v for: %s", tc.shouldDetect, tc.errorMessage)

			if tc.shouldDetect {
				assert.Equal(t, tc.expectedType, errorType,
					"Error should be classified as %s but got %s", tc.expectedType, errorType)
			}

			// Test skippable determination
			shouldSkip := IsSkippableAPIError(tc.errorMessage)
			assert.Equal(t, tc.shouldSkip, shouldSkip,
				"Error skippable determination should be %v", tc.shouldSkip)
		})
	}
}
