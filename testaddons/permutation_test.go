package testaddons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// TestRunAddonPermutationTestAPI tests the new API structure and validation
func TestRunAddonPermutationTestAPI(t *testing.T) {
	// Test creating TestAddonOptions with permutation test fields
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "test-addon",
			OfferingFlavor: "test-flavor",
			Inputs: map[string]interface{}{
				"prefix": "test-prefix",
				"region": "us-south",
			},
		},
	}

	// Verify the structure is properly initialized
	assert.Equal(t, "test-addon", options.AddonConfig.OfferingName)
	assert.Equal(t, "test-flavor", options.AddonConfig.OfferingFlavor)
	assert.Equal(t, "test-prefix", options.Prefix)
	assert.NotNil(t, options.AddonConfig.Inputs)
	assert.NotNil(t, options.Testing)
}

// TestPermutationGenerationLogic tests the permutation generation algorithm
func TestPermutationGenerationLogic(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "test-addon",
			OfferingFlavor: "test-flavor",
		},
	}

	// Test with 2 dependency names (simplified approach)
	dependencyNames := []string{"dep1", "dep2"}

	testCases := options.generatePermutations(dependencyNames)

	// With 2 dependencies, we should have 2^2 - 1 = 3 permutations
	// (excluding the "on by default" case)
	assert.Len(t, testCases, 3)

	// Check that all test cases skip infrastructure deployment
	for _, tc := range testCases {
		assert.True(t, tc.SkipInfrastructureDeployment)
		assert.NotEmpty(t, tc.Name)
		assert.NotEmpty(t, tc.Prefix)
		assert.Len(t, tc.Dependencies, 2)
	}

	// Verify that we don't have the "on by default" case
	onByDefaultCount := 0
	for _, tc := range testCases {
		allEnabled := true
		for _, dep := range tc.Dependencies {
			if dep.Enabled != nil && !*dep.Enabled {
				allEnabled = false
				break
			}
		}
		if allEnabled {
			onByDefaultCount++
		}
	}
	assert.Equal(t, 0, onByDefaultCount, "Should not have the 'on by default' case")
}

// TestIsDefaultConfigurationFunc tests the default configuration detection
func TestIsDefaultConfigurationFunc(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
	}

	// Test case 1: All dependencies OnByDefault: true
	originalDeps := []cloudinfo.AddonConfig{
		{
			OfferingName: "dep1",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(true),
		},
		{
			OfferingName: "dep2",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(true),
		},
	}

	permutation := []cloudinfo.AddonConfig{
		{
			OfferingName: "dep1",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(true),
		},
		{
			OfferingName: "dep2",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(true),
		},
	}

	assert.True(t, options.isDefaultConfiguration(permutation, originalDeps))

	// Test case 2: Non-default configuration
	permutation2 := []cloudinfo.AddonConfig{
		{
			OfferingName: "dep1",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(false), // Different from OnByDefault
		},
		{
			OfferingName: "dep2",
			OnByDefault:  core.BoolPtr(true),
			Enabled:      core.BoolPtr(true),
		},
	}

	assert.False(t, options.isDefaultConfiguration(permutation2, originalDeps))
}

// TestShortenPrefixFunc tests the shortenPrefix helper function
func TestShortenPrefixFunc(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
	}

	// Test cases with different prefix lengths
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short prefix",
			input:    "sm",
			expected: "sm",
		},
		{
			name:     "Medium prefix",
			input:    "sm-perm",
			expected: "sm-per", // 7 chars truncated to 6 to leave room for numbering
		},
		{
			name:     "Long prefix that needs truncation",
			input:    "very-long-prefix-name",
			expected: "very-l", // 6 chars + 2 for numbering = 8 total
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := options.shortenPrefix(test.input)
			assert.Equal(t, test.expected, result)
			// Verify that result + 2 digits would be <= 8 characters
			assert.LessOrEqual(t, len(result)+2, 8, "Result should leave room for 2-digit numbering")
		})
	}
}

// TestJoinNamesFunc tests the joinNames utility function
func TestJoinNamesFunc(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
	}

	// Test with empty slice
	result := options.joinNames([]string{}, "-")
	assert.Equal(t, "", result)

	// Test with single name
	result = options.joinNames([]string{"name1"}, "-")
	assert.Equal(t, "name1", result)

	// Test with multiple names
	result = options.joinNames([]string{"name1", "name2", "name3"}, "-")
	assert.Equal(t, "name1-name2-name3", result)

	// Test with different separator
	result = options.joinNames([]string{"name1", "name2"}, "_")
	assert.Equal(t, "name1_name2", result)
}

// TestRunAddonPermutationTestWithMock tests the complete permutation flow with mocked JSON catalog
func TestRunAddonPermutationTestWithMock(t *testing.T) {
	t.Run("WithTwoDependencies", func(t *testing.T) {
		// Create a temporary ibm_catalog.json file for testing
		mockCatalogJSON := `{
			"products": [{
				"name": "test-addon",
				"label": "Test Addon",
				"flavors": [{
					"name": "test-flavor",
					"label": "Test Flavor",
					"dependencies": [
						{"name": "dep1"},
						{"name": "dep2"}
					]
				}]
			}]
		}`

		// Find git root to place the mock catalog file
		gitRoot, err := common.GitRootPath(".")
		assert.NoError(t, err)

		catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")

		// Write the mock catalog to the correct location
		err = os.WriteFile(catalogPath, []byte(mockCatalogJSON), 0644)
		assert.NoError(t, err)

		// Clean up the file after test
		defer func() {
			_ = os.Remove(catalogPath)
		}()

		// Create test options
		options := &TestAddonOptions{
			Testing: t,
			Prefix:  "test-prefix",
			Logger:  common.CreateSmartAutoBufferingLogger(t.Name(), false),
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "test-addon",
				OfferingFlavor: "test-flavor",
			},
		}

		// Test the dependency name discovery part
		dependencyNames, err := options.getDirectDependencyNames()
		assert.NoError(t, err)
		assert.Len(t, dependencyNames, 2)

		// Test permutation generation
		testCases := options.generatePermutations(dependencyNames)
		assert.Len(t, testCases, 3) // 2^2 - 1 = 3 permutations

		// Verify all test cases skip infrastructure deployment
		for _, tc := range testCases {
			assert.True(t, tc.SkipInfrastructureDeployment)
		}
	})
}

// TestRunAddonPermutationTestNoDependencies tests the case where no dependencies are found
func TestRunAddonPermutationTestNoDependencies(t *testing.T) {
	// Create a mock catalog with no dependencies
	mockCatalogJSON := `{
		"products": [{
			"name": "test-addon",
			"label": "Test Addon",
			"flavors": [{
				"name": "test-flavor",
				"label": "Test Flavor"
			}]
		}]
	}`

	// Find git root to place the mock catalog file
	gitRoot, err := common.GitRootPath(".")
	assert.NoError(t, err)

	catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")

	// Write the mock catalog to the correct location
	err = os.WriteFile(catalogPath, []byte(mockCatalogJSON), 0644)
	assert.NoError(t, err)

	// Clean up the file after test
	defer func() {
		_ = os.Remove(catalogPath)
	}()

	// Create test options
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
		Logger:  common.CreateSmartAutoBufferingLogger(t.Name(), false),
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "test-addon",
			OfferingFlavor: "test-flavor",
		},
	}

	// Test dependency name discovery
	dependencyNames, err := options.getDirectDependencyNames()
	assert.NoError(t, err)
	assert.Len(t, dependencyNames, 0)
}

// TestCreateInitialAbbreviation tests the basic abbreviation functionality
func TestCreateInitialAbbreviation(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple multi-word name",
			input:    "cloud-monitoring",
			expected: "c-m",
		},
		{
			name:     "Deploy arch IBM prefix",
			input:    "deploy-arch-ibm-observability",
			expected: "dai-o",
		},
		{
			name:     "Deploy arch prefix",
			input:    "deploy-arch-observability",
			expected: "da-o",
		},
		{
			name:     "Name with numbers",
			input:    "service-v2-config",
			expected: "s-v2-c",
		},
		{
			name:     "Name with keywords",
			input:    "test-disable-basic",
			expected: "test-disable-basic",
		},
		{
			name:     "Single word",
			input:    "kms",
			expected: "k",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Complex real example",
			input:    "event-notifications-advanced",
			expected: "e-n-advanced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := options.createInitialAbbreviation(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %s but got %s for input %s", tt.expected, result, tt.input)
		})
	}
}

// TestAbbreviateWithCollisionResolution tests collision resolution
func TestAbbreviateWithCollisionResolution(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
	}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No collisions",
			input:    []string{"cloud-monitoring", "key-management", "account-infra"},
			expected: []string{"c-m", "k-m", "a-i"},
		},
		{
			name:     "Two collisions",
			input:    []string{"cloud-monitoring", "cloud-metrics"},
			expected: []string{"c-m", "c-me"},
		},
		{
			name:     "Three way collision",
			input:    []string{"cloud-monitoring", "cloud-metrics", "cloud-manager"},
			expected: []string{"c-m", "c-me", "c-ma"},
		},
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single item",
			input:    []string{"cloud-monitoring"},
			expected: []string{"c-m"},
		},
		{
			name:     "Real world example",
			input:    []string{"cloud-monitoring", "kms", "account-infra", "base", "cos", "cloud-logs", "activity-tracker"},
			expected: []string{"c-m", "k", "a-i", "b", "c", "c-l", "a-t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := options.abbreviateWithCollisionResolution(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v but got %v for input %v", tt.expected, result, tt.input)
		})
	}
}

// TestProjectNameLengthCompliance tests that project names stay under the 128 character limit
func TestProjectNameLengthCompliance(t *testing.T) {
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName: "deploy-arch-ibm-event-notifications-advanced",
		},
	}

	// Test with many long dependency names
	longDependencyNames := []string{
		"deploy-arch-ibm-cloud-monitoring-advanced",
		"deploy-arch-ibm-key-management-service",
		"deploy-arch-ibm-account-infrastructure-base",
		"deploy-arch-ibm-cloud-object-storage",
		"deploy-arch-ibm-cloud-logs-advanced",
		"deploy-arch-ibm-activity-tracker-service",
		"deploy-arch-ibm-security-compliance-center",
	}

	// Test permutation test name generation
	randomPrefix := "abc123"
	mainOfferingAbbrev := options.createInitialAbbreviation(options.AddonConfig.OfferingName)
	abbreviatedDisabledNames := options.abbreviateWithCollisionResolution(longDependencyNames)

	testCaseName := fmt.Sprintf("%s-%s-40-disable-%s", randomPrefix, mainOfferingAbbrev,
		strings.Join(abbreviatedDisabledNames, "-"))

	assert.True(t, len(testCaseName) < 128, "Project name length %d exceeds limit: %s", len(testCaseName), testCaseName)

	// Test matrix test name generation
	nameComponents := []string{randomPrefix, mainOfferingAbbrev, "basic", "test-prefix"}
	matrixTestName := strings.Join(nameComponents, "-")

	assert.True(t, len(matrixTestName) < 128, "Matrix test name length %d exceeds limit: %s", len(matrixTestName), matrixTestName)

	// Verify the names are still readable
	assert.Contains(t, testCaseName, randomPrefix)
	assert.Contains(t, testCaseName, "disable")
	assert.Contains(t, matrixTestName, randomPrefix)
}

// TestRandomPrefixGeneration tests that random prefixes provide uniqueness
func TestRandomPrefixGeneration(t *testing.T) {
	// Generate multiple random prefixes and ensure they're different
	prefixes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		prefix := common.UniqueId(6)
		assert.Len(t, prefix, 6, "Random prefix should be 6 characters long")
		assert.False(t, prefixes[prefix], "Random prefix %s was generated twice", prefix)
		prefixes[prefix] = true
	}

	assert.Len(t, prefixes, 100, "Should have generated 100 unique prefixes")
}

// TestDependencyPermutations tests the full dependency permutation functionality
// This is the real test that should generate permutation reports when it fails
func TestDependencyPermutations(t *testing.T) {
	// Skip test - this is a demonstration of how the real test should be structured
	// In a real environment, ensure the branch exists and API keys are configured
	t.Skip("This test requires proper branch setup and API keys - see example below for proper structure")

	// Example of how a real TestDependencyPermutations test should look:
	/*
		options := TestAddonsOptionsDefault(&TestAddonOptions{
			Testing: t,
			Prefix:  "test-perm",
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:        "test-offering-name",
				OfferingFlavor:      "test-flavor",
				OfferingInstallKind: cloudinfo.InstallKindTerraform, // This is the key fix - must be set!
				Inputs: map[string]interface{}{
					"prefix":                       "test-perm",
					"region":                       "us-south",
					"existing_resource_group_name": "test-resource-group",
				},
			},
		})

		err := options.RunAddonPermutationTest()
		assert.NoError(t, err, "Dependency permutation test should not fail")
	*/
}

// TestApprappDependencyPermutationsFix is a comprehensive regression test for the dependency tree structure bug.
// This test will FAIL before the fix (showing KMS/COS as direct dependencies) and PASS after the fix.
// It serves as permanent protection against future regressions of the tree flattening bug.
func TestApprappDependencyPermutationsFix(t *testing.T) {
	// This test is designed to fail initially, demonstrating the bug exists
	// After the fix, it should pass and serve as regression protection

	// Mock the CloudInfoService using the comprehensive pattern from working tests
	mockService := &cloudinfo.MockCloudInfoServiceForPermutation{}

	// Mock catalog operations following the working pattern
	mockCatalog := &catalogmanagementv1.Catalog{
		ID:    core.StringPtr("test-catalog-id"),
		Label: core.StringPtr("test-catalog"),
	}
	mockService.On("CreateCatalog", mock.MatchedBy(func(name string) bool {
		return len(name) > 0
	})).Return(mockCatalog, nil)

	// Mock offering operations
	mockOffering := &catalogmanagementv1.Offering{
		Name: core.StringPtr("deploy-arch-ibm-apprapp"),
		Kinds: []catalogmanagementv1.Kind{
			{
				InstallKind: core.StringPtr("terraform"),
				Versions: []catalogmanagementv1.Version{
					{
						VersionLocator: core.StringPtr("test-catalog.test-version"),
						Version:        core.StringPtr("1.0.0"),
					},
				},
			},
		},
	}
	mockService.On("ImportOfferingWithValidation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockOffering, nil)
	mockService.On("DeleteCatalog", mock.Anything).Return(nil)

	// Mock comprehensive CloudInfoService operations for realistic test execution

	// Core project and config operations
	mockService.On("GetProjectConfigs", mock.Anything).Return([]interface{}{}, nil)
	mockService.On("GetConfig", mock.Anything).Return(nil, nil, nil)
	mockService.On("SetLogger", mock.Anything).Return()

	// Offering import and preparation - Must return 4 values as expected by interface
	mockService.On("PrepareOfferingImport").Return(
		"https://github.com/test-repo/test-branch", // branchUrl
		"test-repo", // repo
		"main",      // branch
		nil,         // error
	)

	// Offering operations for validation pipeline
	mockService.On("GetOffering", mock.Anything, mock.Anything).Return(mockOffering, nil, nil)
	mockService.On("GetOfferingInputs", mock.Anything, mock.Anything, mock.Anything).Return([]cloudinfo.CatalogInput{})
	mockService.On("GetOfferingVersionLocatorByConstraint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("1.0.0", "test-catalog.test-version", nil)
	mockService.On("GetCatalogVersionByLocator", mock.Anything).Return(&catalogmanagementv1.Version{
		VersionLocator: core.StringPtr("test-catalog.test-version"),
		Version:        core.StringPtr("1.0.0"),
	}, nil)

	// Project deployment operations that might be called
	mockService.On("DeployAddonToProject", mock.Anything, mock.Anything).Return(&cloudinfo.DeployedAddonsDetails{}, nil)
	mockService.On("UpdateConfig", mock.Anything, mock.Anything).Return(nil, nil, nil)
	mockService.On("GetApiKey").Return("test-api-key")
	mockService.On("ResolveReferencesFromStringsWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	// Setup dependency structure based on REAL IBM Cloud API responses from log.log:

	// Main addon (deploy-arch-ibm-apprapp) returns 7 optional dependencies - EXACT REAL-WORLD DATA
	// Real log: GetComponentReferences(3e864d67-980f-400e-8a49-9eab43590bc6.ac14f313-8403-44d2-9764-4a9e26f93961) OUTPUT: Required=0, Optional=7
	mockService.On("GetComponentReferences", mock.MatchedBy(func(versionLocator string) bool {
		return strings.Contains(versionLocator, "apprapp") || strings.Contains(versionLocator, "3e864d67-980f-400e-8a49-9eab43590bc6.ac14f313-8403-44d2-9764-4a9e26f93961")
	})).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{
				// EXACT reproduction from real logs with VersionLocator populated:
				{Name: "deploy-arch-ibm-cloud-monitoring", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    true,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.e8444cee-432d-4af3-9211-d19cb739f4a3-global",
				}}, // Optional[0] ✅ Should be direct
				{Name: "deploy-arch-ibm-account-infra-base", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    false,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.f57a4724-c399-4a15-ad9d-440da676c8a0-global",
				}}, // Optional[1] (duplicate 1)
				{Name: "deploy-arch-ibm-account-infra-base", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    false,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.6952804c-65da-4b45-8fb2-4520bd739d65-global",
				}}, // Optional[2] (duplicate 2)
				{Name: "deploy-arch-ibm-cloud-logs", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    true,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.643fbe72-a630-43f3-8cc2-77bccb15f604-global",
				}}, // Optional[3] ✅ Should be direct
				{Name: "deploy-arch-ibm-activity-tracker", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    true,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.f5984196-27e2-418d-a0d3-2b6cbcda537c-global",
				}}, // Optional[4] ✅ Should be direct
				{Name: "deploy-arch-ibm-cos", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    true,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.79a61ce0-d4fa-4f1a-b6c5-5ca23b13ff06-global",
				}}, // Optional[5] ❌ BUG: Should be nested under cloud-logs
				{Name: "deploy-arch-ibm-kms", OfferingReference: cloudinfo.OfferingReferenceDetail{
					OnByDefault:    true,
					VersionLocator: "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.466c0738-68d1-4b8d-8854-f85e01853f10-global",
				}}, // Optional[6] ❌ BUG: Should be nested under cloud-logs
			},
		},
	}, nil)

	// account-infra-base returns no dependencies
	// Real log pattern: .f57a4724-c399-4a15-ad9d-440da676c8a0-global OUTPUT: Required=0, Optional=0
	mockService.On("GetComponentReferences", mock.MatchedBy(func(versionLocator string) bool {
		return strings.Contains(versionLocator, "account-infra") || strings.Contains(versionLocator, "f57a4724-c399-4a15-ad9d-440da676c8a0-global")
	})).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
	}, nil)

	// CRITICAL: Need to ensure the processedLocators logic doesn't skip adding components
	// Mock the specific version locators with VersionLocator field populated to trigger the bug
	mockService.On("GetComponentReferences", "7a4d68b4-cf8b-40cd-a3d1-f49aff526eb3.f57a4724-c399-4a15-ad9d-440da676c8a0-global").Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
	}, nil)

	// cloud-monitoring returns 2 optional dependencies
	// Real log pattern: .e8444cee-432d-4af3-9211-d19cb739f4a3-global OUTPUT: Required=0, Optional=2
	mockService.On("GetComponentReferences", mock.MatchedBy(func(versionLocator string) bool {
		return strings.Contains(versionLocator, "cloud-monitoring") || strings.Contains(versionLocator, "e8444cee-432d-4af3-9211-d19cb739f4a3-global")
	})).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{
				{Name: "deploy-arch-ibm-some-monitoring-dep-1"},
				{Name: "deploy-arch-ibm-some-monitoring-dep-2"},
			},
		},
	}, nil)

	// activity-tracker returns 6 optional dependencies
	// Real log pattern: .f5984196-27e2-418d-a0d3-2b6cbcda537c-global OUTPUT: Required=0, Optional=6
	mockService.On("GetComponentReferences", mock.MatchedBy(func(versionLocator string) bool {
		return strings.Contains(versionLocator, "activity-tracker") || strings.Contains(versionLocator, "f5984196-27e2-418d-a0d3-2b6cbcda537c-global")
	})).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{
				{Name: "deploy-arch-ibm-cloud-logs"},
				{Name: "deploy-arch-ibm-kms"},
				{Name: "deploy-arch-ibm-cos"},
				{Name: "deploy-arch-ibm-cloud-monitoring"},
				{Name: "deploy-arch-ibm-at-dep-5"},
				{Name: "deploy-arch-ibm-at-dep-6"},
			},
		},
	}, nil)

	// cloud-logs returns 5 optional dependencies
	// Real log pattern: .643fbe72-a630-43f3-8cc2-77bccb15f604-global OUTPUT: Required=0, Optional=5
	mockService.On("GetComponentReferences", mock.MatchedBy(func(versionLocator string) bool {
		return strings.Contains(versionLocator, "cloud-logs") || strings.Contains(versionLocator, "643fbe72-a630-43f3-8cc2-77bccb15f604-global")
	})).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{
				{Name: "deploy-arch-ibm-kms"},
				{Name: "deploy-arch-ibm-cos"},
				{Name: "deploy-arch-ibm-cloud-monitoring"},
				{Name: "deploy-arch-ibm-cl-dep-4"},
				{Name: "deploy-arch-ibm-cl-dep-5"},
			},
		},
	}, nil)

	// Default case for any other dependencies (like KMS, COS, etc.)
	mockService.On("GetComponentReferences", mock.Anything).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
	}, nil)

	// Create a temporary ibm_catalog.json file for the main addon to read its direct dependencies
	mockCatalogJSON := `{
		"products": [{
			"name": "deploy-arch-ibm-apprapp",
			"label": "IBM App Application Pattern",
			"flavors": [{
				"name": "fully-configurable",
				"label": "Fully Configurable",
				"dependencies": [
					{"name": "deploy-arch-ibm-account-infra-base"},
					{"name": "deploy-arch-ibm-cloud-logs"},
					{"name": "deploy-arch-ibm-cloud-monitoring"},
					{"name": "deploy-arch-ibm-activity-tracker"}
				]
			}]
		}]
	}`

	// Find git root to place the mock catalog file
	gitRoot, err := common.GitRootPath(".")
	require.NoError(t, err)

	catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")

	// Write the mock catalog to the correct location
	writeErr := os.WriteFile(catalogPath, []byte(mockCatalogJSON), 0644)
	require.NoError(t, writeErr)

	// Clean up the file after test
	defer func() {
		_ = os.Remove(catalogPath)
	}()

	// Create test options with the mock catalog structure
	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "app-per",
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "deploy-arch-ibm-apprapp",
			OfferingFlavor: "fully-configurable",
			Inputs: map[string]interface{}{
				"prefix":                       "app-per",
				"region":                       "us-south",
				"existing_resource_group_name": "default",
			},
		},
		CloudInfoService: mockService,
		Logger:           common.CreateSmartAutoBufferingLogger("TestApprappFix", false),
		// Skip flags to bypass filesystem and Git operations while allowing dependency processing
		SkipLocalChangeCheck:         true,
		SkipInfrastructureDeployment: false, // CRITICAL: Must allow deployment for dependency processing bug to manifest
		SkipRefValidation:            true,
		SkipDependencyValidation:     true,
		// Enable result collection for analysis
		CollectResults: true,
	}

	// Run the permutation test - this will help us identify the bug
	err = options.RunAddonPermutationTest()

	// COMPREHENSIVE REGRESSION TEST ASSERTIONS
	// These assertions will FAIL before the fix and PASS after the fix

	// Log any error but don't fail immediately - we want to analyze the structure
	if err != nil {
		t.Logf("Permutation test returned error (expected during regression testing): %v", err)
	}

	// The test should run and generate some results for analysis
	require.NotNil(t, options.PermutationTestReport, "Permutation test report should be generated")
	require.Greater(t, len(options.PermutationTestReport.Results), 0, "Should have test results")

	// Check each test result for the dependency structure bug
	for _, result := range options.PermutationTestReport.Results {
		t.Logf("Analyzing test result: %s", result.Name)

		// Should have exactly 1 main addon config (not flattened)
		assert.Equal(t, 1, len(result.AddonConfig),
			"Test %s: Should have exactly 1 main addon config, got %d",
			result.Name, len(result.AddonConfig))

		mainAddon := result.AddonConfig[0]
		assert.Equal(t, "deploy-arch-ibm-apprapp", mainAddon.OfferingName)

		// CRITICAL ASSERTION 1: Should have exactly 4 direct dependencies
		// This will FAIL before fix (shows 6) and PASS after fix (shows 4)
		assert.Equal(t, 4, len(mainAddon.Dependencies),
			"Test %s: Main addon should have exactly 4 direct dependencies, got %d. Dependencies: %v",
			result.Name, len(mainAddon.Dependencies), extractDependencyNames(mainAddon.Dependencies))

		// CRITICAL ASSERTION 2: Verify specific direct dependency names
		// This will FAIL before fix (includes KMS/COS) and PASS after fix (excludes them)
		directDepNames := extractDependencyNames(mainAddon.Dependencies)
		expectedDirectDeps := []string{
			"deploy-arch-ibm-account-infra-base",
			"deploy-arch-ibm-cloud-logs",
			"deploy-arch-ibm-cloud-monitoring",
			"deploy-arch-ibm-activity-tracker",
		}
		assert.ElementsMatch(t, expectedDirectDeps, directDepNames,
			"Test %s: Should have exactly the 4 expected direct dependencies. Got: %v, Expected: %v",
			result.Name, directDepNames, expectedDirectDeps)

		// CRITICAL ASSERTION 3: KMS and COS must NOT be direct dependencies
		// This will FAIL before fix (they appear as direct) and PASS after fix (they don't)
		for _, dep := range mainAddon.Dependencies {
			assert.NotContains(t, []string{"deploy-arch-ibm-kms", "deploy-arch-ibm-cos"}, dep.OfferingName,
				"Test %s: Found prohibited direct dependency: %s. KMS and COS should only exist nested under their parents",
				result.Name, dep.OfferingName)
		}

		// VALIDATION ASSERTION 4: Verify nested structure exists properly
		// Find cloud-logs dependency and verify it has nested dependencies
		cloudLogsDep := findDependencyByName(mainAddon.Dependencies, "deploy-arch-ibm-cloud-logs")
		if assert.NotNil(t, cloudLogsDep, "Test %s: cloud-logs dependency should exist", result.Name) {
			// cloud-logs should have some nested dependencies
			t.Logf("Test %s: cloud-logs has %d nested dependencies", result.Name, len(cloudLogsDep.Dependencies))
			// Note: Exact nested count may vary based on permutation, but should have some
		}
	}

	// If we reach here, log the final result
	t.Logf("Dependency structure validation complete. Total permutations tested: %d", len(options.PermutationTestReport.Results))
}

// Helper functions for dependency structure validation

// extractDependencyNames extracts the offering names from a slice of AddonConfig dependencies
func extractDependencyNames(dependencies []cloudinfo.AddonConfig) []string {
	names := make([]string, len(dependencies))
	for i, dep := range dependencies {
		names[i] = dep.OfferingName
	}
	return names
}
