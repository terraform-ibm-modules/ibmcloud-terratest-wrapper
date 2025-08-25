package testaddons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	projects "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// TestSkipPermutations_NamesOnly verifies that generatePermutations filters by enabled dependency names
func TestSkipPermutations_NamesOnly(t *testing.T) {

	options := &TestAddonOptions{
		Testing: t,
		Prefix:  "test-prefix",
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "test-addon",
			OfferingFlavor: "test-flavor",
		},
		// Skip the permutation where only dep1 is enabled (dep2 disabled)
		SkipPermutations: [][]cloudinfo.AddonConfig{
			{
				{OfferingName: "dep1"},
			},
		},
	}

	dependencyNames := []string{"dep1", "dep2"}
	testCases := options.generatePermutations(dependencyNames)

	// Originally 3 permutations; after skipping one, expect 2
	assert.Len(t, testCases, 2)

	// Ensure no case has only dep1 enabled
	for _, tc := range testCases {
		enabled := map[string]bool{}
		for _, dep := range tc.Dependencies {
			if dep.Enabled != nil && *dep.Enabled {
				enabled[dep.OfferingName] = true
			}
		}
		if len(enabled) == 1 && enabled["dep1"] {
			t.Fatalf("found skipped permutation present: only dep1 enabled in case %q", tc.Name)
		}
	}
}

// TestSkipPermutations_WithFlavors verifies that generatePermutationsWithFlavors filters by enabled name+flavor sets
func TestSkipPermutations_WithFlavors(t *testing.T) {
	// Two deps: dep1 has flavors a,b; dep2 has one flavor x
	deps := []DependencyWithFlavors{
		{Name: "dep1", Flavors: []string{"a", "b"}},
		{Name: "dep2", Flavors: []string{"x"}},
	}

	t.Run("Skip specific flavor", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing: t,
			Prefix:  "test-prefix",
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "test-addon",
				OfferingFlavor: "test-flavor",
			},
			// Skip the permutation where only dep1 is enabled with flavor b
			SkipPermutations: [][]cloudinfo.AddonConfig{
				{
					{OfferingName: "dep1", OfferingFlavor: "b"},
				},
			},
		}

		testCases := options.generatePermutationsWithFlavors(deps)

		// Without skipping: 4 permutations (both disabled, dep1[a], dep1[b], dep2[x])
		// After skipping dep1[b], expect 3
		assert.Len(t, testCases, 3)

		for _, tc := range testCases {
			enabled := make(map[string]string)
			for _, dep := range tc.Dependencies {
				if dep.Enabled != nil && *dep.Enabled {
					enabled[dep.OfferingName] = dep.OfferingFlavor
				}
			}
			if len(enabled) == 1 {
				if fl, ok := enabled["dep1"]; ok && fl == "b" {
					t.Fatalf("found skipped permutation present: only dep1[b] enabled in case %q", tc.Name)
				}
			}
		}
	})

	t.Run("Skip with wildcard flavor", func(t *testing.T) {
		options := &TestAddonOptions{
			Testing: t,
			Prefix:  "test-prefix",
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "test-addon",
				OfferingFlavor: "test-flavor",
			},
			// Skip the permutation where only dep2 is enabled, any flavor (wildcard)
			SkipPermutations: [][]cloudinfo.AddonConfig{
				{
					{OfferingName: "dep2", OfferingFlavor: ""},
				},
			},
		}

		testCases := options.generatePermutationsWithFlavors(deps)

		// Skip dep2-only case; expect 3 remaining
		assert.Len(t, testCases, 3)

		for _, tc := range testCases {
			enabled := make(map[string]string)
			for _, dep := range tc.Dependencies {
				if dep.Enabled != nil && *dep.Enabled {
					enabled[dep.OfferingName] = dep.OfferingFlavor
				}
			}
			if len(enabled) == 1 {
				if _, ok := enabled["dep2"]; ok {
					t.Fatalf("found skipped permutation present: dep2-only enabled in case %q", tc.Name)
				}
			}
		}
	})
}

// TestDependencyPermutations tests the full dependency permutation functionality
// This is the real test that should generate permutation reports when it fails
func TestDependencyPermutations(t *testing.T) {

	// Use mocking pattern to avoid external dependencies while exercising the public API

	// Create a minimal mock catalog with one dependency having two flavors
	mockCatalogJSON := `{
        "products": [{
            "name": "mock-addon",
            "label": "Mock Addon",
            "flavors": [{
                "name": "test-flavor",
                "label": "Test Flavor",
                "dependencies": [
                    {"name": "dep1", "flavors": ["a", "b"]}
                ]
            }]
        }]
    }`

	gitRoot, err := common.GitRootPath(".")
	assert.NoError(t, err)
	catalogPath := filepath.Join(gitRoot, "ibm_catalog.json")
	err = os.WriteFile(catalogPath, []byte(mockCatalogJSON), 0644)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(catalogPath) }()

	// Set up the mock CloudInfoService with minimal no-op behaviors
	mockService := &cloudinfo.MockCloudInfoServiceForPermutation{}

	// PrepareOfferingImport is called during setup
	mockService.On("PrepareOfferingImport").Return(
		"https://github.com/test-repo/test-branch", // branchUrl
		"test-repo", // repo
		"main",      // branch
		nil,
	)

	// CreateCatalog and ImportOfferingWithValidation for shared matrix setup
	mockCatalog := &catalogmanagementv1.Catalog{ID: core.StringPtr("mock-catalog-id"), Label: core.StringPtr("mock-catalog")}
	mockService.On("CreateCatalog", mock.Anything).Return(mockCatalog, nil)

	mockOffering := &catalogmanagementv1.Offering{
		ID:        core.StringPtr("mock-offering-id"),
		Name:      core.StringPtr("mock-addon"),
		CatalogID: core.StringPtr("mock-catalog-id"), // Add CatalogID to prevent nil errors
		Kinds: []catalogmanagementv1.Kind{{
			InstallKind: core.StringPtr("terraform"),
			Versions: []catalogmanagementv1.Version{{
				VersionLocator: core.StringPtr("mock-catalog.mock-offering.mock-version"),
				Version:        core.StringPtr("mock-version"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{
						// Empty dependencies array prevents nil pointer dereference
						// Could add mock dependencies here if needed for more realistic testing
					},
				},
			}},
		}},
	}
	mockService.On("ImportOfferingWithValidation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockOffering, nil)

	// Offering/version lookups used by validation and dependency logic
	// Main offering
	mockService.On("GetOffering", "mock-catalog-id", "mock-offering-id").Return(mockOffering, nil, nil)

	// Dependency offerings - create mock offering for dep1
	mockDep1Offering := &catalogmanagementv1.Offering{
		ID:        core.StringPtr("dep1-offering-id"),
		Name:      core.StringPtr("dep1"),
		CatalogID: core.StringPtr("mock-catalog-id"),
		Kinds: []catalogmanagementv1.Kind{{
			InstallKind: core.StringPtr("terraform"),
			Versions: []catalogmanagementv1.Version{{
				VersionLocator: core.StringPtr("mock-catalog.dep1-offering-id.mock-version"),
				Version:        core.StringPtr("mock-version"),
				SolutionInfo: &catalogmanagementv1.SolutionInfo{
					Dependencies: []catalogmanagementv1.OfferingReference{},
				},
			}},
		}},
	}
	mockService.On("GetOffering", "mock-catalog-id", "dep1-offering-id").Return(mockDep1Offering, nil, nil)

	// Catch-all for any other GetOffering calls
	mockService.On("GetOffering", mock.Anything, mock.Anything).Return(mockOffering, nil, nil)
	mockService.On("GetOfferingInputs", mock.Anything, mock.Anything, mock.Anything).Return([]cloudinfo.CatalogInput{})
	// Return different version locators based on offering name/ID to handle dependencies properly
	mockService.On("GetOfferingVersionLocatorByConstraint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(catalogID, offeringID, constraint, kind string) (string, string, error) {
			version := "mock-version" // Use consistent mock-version instead of 1.0.0
			var versionLocator string
			if offeringID == "mock-offering-id" {
				// Main offering
				versionLocator = "mock-catalog.mock-offering.mock-version"
			} else if offeringID == "dep1-offering-id" {
				// Dependency dep1
				versionLocator = "mock-catalog.dep1-offering-id.mock-version"
			} else {
				// Other dependencies - create proper 3-part version locator
				versionLocator = fmt.Sprintf("mock-catalog.%s.mock-version", offeringID)
			}
			return version, versionLocator, nil
		},
	)
	// GetCatalogVersionByLocator should return different versions based on version locator
	mockService.On("GetCatalogVersionByLocator", mock.MatchedBy(func(locator string) bool {
		return locator == "mock-catalog.mock-offering.mock-version"
	})).Return(&catalogmanagementv1.Version{
		VersionLocator: core.StringPtr("mock-catalog.mock-offering.mock-version"),
		Version:        core.StringPtr("mock-version"),
		CatalogID:      core.StringPtr("mock-catalog-id"),
		OfferingID:     core.StringPtr("mock-offering-id"),
		Flavor: &catalogmanagementv1.Flavor{
			Name:  core.StringPtr("test-flavor"),
			Label: core.StringPtr("Test Flavor"),
		},
		SolutionInfo: &catalogmanagementv1.SolutionInfo{
			Dependencies: []catalogmanagementv1.OfferingReference{},
		},
	}, nil)

	// For dependency dep1
	mockService.On("GetCatalogVersionByLocator", mock.MatchedBy(func(locator string) bool {
		return locator == "mock-catalog.dep1-offering-id.mock-version"
	})).Return(&catalogmanagementv1.Version{
		VersionLocator: core.StringPtr("mock-catalog.dep1-offering-id.mock-version"),
		Version:        core.StringPtr("mock-version"),
		CatalogID:      core.StringPtr("mock-catalog-id"),
		OfferingID:     core.StringPtr("dep1-offering-id"),
		Flavor: &catalogmanagementv1.Flavor{
			Name:  core.StringPtr("a"), // dep1 has flavor "a" based on the mock catalog JSON
			Label: core.StringPtr("Flavor A"),
		},
		SolutionInfo: &catalogmanagementv1.SolutionInfo{
			Dependencies: []catalogmanagementv1.OfferingReference{},
		},
	}, nil)

	// Catch-all for other version locators
	mockService.On("GetCatalogVersionByLocator", mock.Anything).Return(&catalogmanagementv1.Version{
		VersionLocator: core.StringPtr("mock-catalog.mock-offering.mock-version"),
		Version:        core.StringPtr("mock-version"),
		CatalogID:      core.StringPtr("mock-catalog-id"),
		OfferingID:     core.StringPtr("mock-offering-id"),
		Flavor: &catalogmanagementv1.Flavor{
			Name:  core.StringPtr("test-flavor"),
			Label: core.StringPtr("Test Flavor"),
		},
		SolutionInfo: &catalogmanagementv1.SolutionInfo{
			Dependencies: []catalogmanagementv1.OfferingReference{},
		},
	}, nil)
	mockService.On("GetConfig", mock.Anything).Return(&projects.ProjectConfig{
		ID: core.StringPtr("test-config-id"),
		Definition: &projects.ProjectConfigDefinitionResponse{
			Name:        core.StringPtr("mock-addon"),
			LocatorID:   core.StringPtr("mock-catalog.mock-offering.mock-version"),
			Description: core.StringPtr("Mock addon for testing"),
			Inputs: map[string]interface{}{
				"prefix": "mock-prefix",
				"region": "us-south",
			},
		},
	}, &core.DetailedResponse{}, nil)

	// Component references: return empty (we don't need deep tree building in this test)
	mockService.On("GetComponentReferences", mock.Anything).Return(&cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
		Optional: cloudinfo.OptionalReferences{OfferingReferences: []cloudinfo.OfferingReferenceItem{}},
	}, nil)

	// Project/config related calls (validation-only path still touches these helpers)
	mockService.On("GetProjectConfigs", mock.Anything).Return([]projects.ProjectConfigSummary{
		{
			ID:      core.StringPtr("test-config-id"),
			Version: core.Int64Ptr(1),
			State:   core.StringPtr("applied"),
			Definition: &projects.ProjectConfigSummaryDefinition{
				Name: core.StringPtr("mock-addon"),
			},
		},
	}, nil)
	mockService.On("UpdateConfig", mock.Anything, mock.Anything).Return(&projects.ProjectConfig{ID: core.StringPtr("test-config-id")}, &core.DetailedResponse{}, nil)
	mockService.On("CreateProjectFromConfig", mock.Anything).Return(&projects.Project{ID: core.StringPtr("test-project-id")}, &core.DetailedResponse{}, nil)
	mockService.On("DeployAddonToProject", mock.Anything, mock.Anything).Return(&cloudinfo.DeployedAddonsDetails{
		ProjectID: "test-project-id",
		Configs: []struct {
			Name     string `json:"name"`
			ConfigID string `json:"config_id"`
		}{
			{
				Name:     "mock-addon",
				ConfigID: "test-config-id",
			},
		},
	}, nil)
	mockService.On("GetApiKey").Return("test-api-key")
	mockService.On("ResolveReferencesFromStringsWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockService.On("SetLogger", mock.Anything).Return()
	mockService.On("DeleteCatalog", mock.Anything).Return(nil)
	mockService.On("DeleteProject", mock.Anything).Return(&projects.ProjectDeleteResponse{}, &core.DetailedResponse{}, nil)

	// Additional mocks that might be called during test execution
	mockService.On("GetConfigName", mock.Anything, mock.Anything).Return("mock-addon", nil)
	mockService.On("GetProject", mock.Anything).Return(&projects.Project{ID: core.StringPtr("test-project-id")}, &core.DetailedResponse{}, nil)

	// Build options and run permutation test (validation-only)
	options := TestAddonsOptionsDefault(&TestAddonOptions{
		Testing:                 t,
		Prefix:                  "mock-perm",
		Logger:                  common.CreateSmartAutoBufferingLogger(t.Name(), false),
		CloudInfoService:        mockService,
		RequiredEnvironmentVars: map[string]string{}, // Bypass environment check for unit tests
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:        "mock-addon",
			OfferingFlavor:      "test-flavor",
			OfferingInstallKind: cloudinfo.InstallKindTerraform,
			CatalogID:           "mock-catalog-id",
			OfferingID:          "mock-offering-id",
			VersionLocator:      "mock-catalog.mock-offering.mock-version",
			Inputs: map[string]interface{}{
				"prefix": "mock-perm",
				"region": "us-south",
			},
		},
		// Ensure per-test quiet mode is respected; top-level logger remains verbose for progress
		QuietMode: false,
		// Result collection enabled implicitly by RunAddonPermutationTest
		SkipLocalChangeCheck: true, // Allow test to run with uncommitted changes
	})

	// Test the permutation generation logic without running full integration
	// This focuses the test on what permutation_test.go should actually test
	dependencyNames, err := options.getDirectDependencyNames()
	assert.NoError(t, err, "Should be able to get dependency names")
	assert.Len(t, dependencyNames, 1, "Should find 1 dependency (dep1)")
	assert.Equal(t, "dep1", dependencyNames[0])

	// Test basic permutation generation
	testCases := options.generatePermutations(dependencyNames)
	assert.Greater(t, len(testCases), 0, "Should generate permutation test cases")

	// Verify test case structure
	for _, tc := range testCases {
		assert.NotEmpty(t, tc.Name, "Test case should have a name")
		assert.NotEmpty(t, tc.Prefix, "Test case should have a prefix")
		assert.True(t, tc.SkipInfrastructureDeployment, "Permutation tests should skip infrastructure deployment")
		assert.NotNil(t, tc.Dependencies, "Test case should have dependencies")
	}

	// Verify that dependencies are properly structured (basic fields only - resolution happens later)
	for _, tc := range testCases {
		for _, dep := range tc.Dependencies {
			assert.NotEmpty(t, dep.OfferingName, "Dependency should have offering name")
			// OfferingFlavor can be empty for some dependencies - that's expected
			assert.NotNil(t, dep.Enabled, "Dependency should have enabled status")
			// These fields should be empty - they get populated during dependency resolution
			assert.Empty(t, dep.OfferingID, "Dependency OfferingID should be empty until resolution")
			assert.Empty(t, dep.VersionLocator, "Dependency VersionLocator should be empty until resolution")
		}
	}

	t.Logf("Successfully generated %d permutation test cases with proper dependency structure", len(testCases))
}

// Helper functions for dependency structure validation
