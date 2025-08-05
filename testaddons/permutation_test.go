package testaddons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/stretchr/testify/assert"
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
