package testaddons

import (
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
	logger := common.NewTestLogger(t.Name())

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
	logger := common.NewTestLogger(t.Name())

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
	logger := common.NewTestLogger(t.Name())

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

// TestValidateDependenciesDetectsMissingConfigs tests that validateDependencies
// properly detects when expected configurations are missing from the deployed list
func TestValidateDependenciesDetectsMissingConfigs(t *testing.T) {
	logger := common.NewTestLogger(t.Name())

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Create a simple dependency graph
	stringGraph := make(map[string][]cloudinfo.OfferingReferenceDetail)

	// Expected deployed list - contains configs that should be deployed
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
	logger := common.NewTestLogger(t.Name())

	options := &TestAddonOptions{
		Testing:          t,
		Logger:           logger,
		CloudInfoService: &MockCloudInfoService{},
	}

	// Create a simple dependency graph
	stringGraph := make(map[string][]cloudinfo.OfferingReferenceDetail)

	// Expected deployed list
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
	logger := common.NewTestLogger(t.Name())

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
	logger := common.NewTestLogger(t.Name())

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
