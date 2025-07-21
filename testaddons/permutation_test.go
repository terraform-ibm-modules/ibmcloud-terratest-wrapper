package testaddons

import (
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
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

	// Test with 2 dependencies
	dependencies := []cloudinfo.AddonConfig{
		{
			OfferingName:   "dep1",
			OfferingFlavor: "flavor1",
			OnByDefault:    core.BoolPtr(true),
			Enabled:        core.BoolPtr(true),
		},
		{
			OfferingName:   "dep2",
			OfferingFlavor: "flavor2",
			OnByDefault:    core.BoolPtr(true),
			Enabled:        core.BoolPtr(true),
		},
	}

	testCases := options.generatePermutations(dependencies)

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

// TestRunAddonPermutationTestWithMock tests the complete permutation flow with mocked dependencies
func TestRunAddonPermutationTestWithMock(t *testing.T) {
	t.Run("WithTwoDependencies", func(t *testing.T) {
		// Create a mock CloudInfoService
		mockService := &cloudinfo.MockCloudInfoServiceForPermutation{}

		// Set up mock expectations
		mockCatalog := &catalogmanagementv1.Catalog{
			ID:    core.StringPtr("test-catalog-id"),
			Label: core.StringPtr("test-catalog"),
		}

		mockOffering := &catalogmanagementv1.Offering{
			Name: core.StringPtr("test-addon"),
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

		mockComponentReferences := &cloudinfo.OfferingReferenceResponse{
			Required: cloudinfo.RequiredReferences{
				OfferingReferences: []cloudinfo.OfferingReferenceItem{},
			},
			Optional: cloudinfo.OptionalReferences{
				OfferingReferences: []cloudinfo.OfferingReferenceItem{
					{
						Name: "dep1",
						OfferingReference: cloudinfo.OfferingReferenceDetail{
							Name:          "dep1",
							Flavor:        cloudinfo.Flavor{Name: "flavor1"},
							OnByDefault:   true,
							DefaultFlavor: "flavor1",
						},
					},
					{
						Name: "dep2",
						OfferingReference: cloudinfo.OfferingReferenceDetail{
							Name:          "dep2",
							Flavor:        cloudinfo.Flavor{Name: "flavor2"},
							OnByDefault:   true,
							DefaultFlavor: "flavor2",
						},
					},
				},
			},
		}

		// Set up mock method expectations
		mockService.On("CreateCatalog", mock.MatchedBy(func(name string) bool {
			return len(name) > 0
		})).Return(mockCatalog, nil)

		mockService.On("ImportOfferingWithValidation",
			"test-catalog-id",
			"test-addon",
			"test-flavor",
			"1.0.0",
			cloudinfo.InstallKindTerraform).Return(mockOffering, nil)

		mockService.On("GetComponentReferences", "test-catalog.test-version").Return(mockComponentReferences, nil)

		mockService.On("DeleteCatalog", "test-catalog-id").Return(nil)

		// Create test options with the mock service
		options := &TestAddonOptions{
			Testing:          t,
			Prefix:           "test-prefix",
			CloudInfoService: mockService,
			Logger:           common.NewTestLogger(t.Name()),
			AddonConfig: cloudinfo.AddonConfig{
				OfferingName:   "test-addon",
				OfferingFlavor: "test-flavor",
			},
		}

		// This should not panic and should not call the matrix test since we're not testing the full flow
		// Instead, let's just test the dependency discovery part
		dependencies, err := options.discoverDependencies()
		assert.NoError(t, err)
		assert.Len(t, dependencies, 2)

		// Test permutation generation
		testCases := options.generatePermutations(dependencies)
		assert.Len(t, testCases, 3) // 2^2 - 1 = 3 permutations

		// Verify all test cases skip infrastructure deployment
		for _, tc := range testCases {
			assert.True(t, tc.SkipInfrastructureDeployment)
		}

		// Verify mock expectations were met
		mockService.AssertExpectations(t)
	})
}

// TestRunAddonPermutationTestNoDependencies tests the case where no dependencies are found
func TestRunAddonPermutationTestNoDependencies(t *testing.T) {
	// Create a mock CloudInfoService
	mockService := &cloudinfo.MockCloudInfoServiceForPermutation{}

	// Set up mock expectations for no dependencies
	mockCatalog := &catalogmanagementv1.Catalog{
		ID:    core.StringPtr("test-catalog-id"),
		Label: core.StringPtr("test-catalog"),
	}

	mockOffering := &catalogmanagementv1.Offering{
		Name: core.StringPtr("test-addon"),
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

	mockComponentReferences := &cloudinfo.OfferingReferenceResponse{
		Required: cloudinfo.RequiredReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{},
		},
		Optional: cloudinfo.OptionalReferences{
			OfferingReferences: []cloudinfo.OfferingReferenceItem{},
		},
	}

	// Set up mock method expectations
	mockService.On("CreateCatalog", mock.MatchedBy(func(name string) bool {
		return len(name) > 0
	})).Return(mockCatalog, nil)

	mockService.On("ImportOfferingWithValidation",
		"test-catalog-id",
		"test-addon",
		"test-flavor",
		"1.0.0",
		cloudinfo.InstallKindTerraform).Return(mockOffering, nil)

	mockService.On("GetComponentReferences", "test-catalog.test-version").Return(mockComponentReferences, nil)

	mockService.On("DeleteCatalog", "test-catalog-id").Return(nil)

	// Create test options with the mock service
	options := &TestAddonOptions{
		Testing:          t,
		Prefix:           "test-prefix",
		CloudInfoService: mockService,
		Logger:           common.NewTestLogger(t.Name()),
		AddonConfig: cloudinfo.AddonConfig{
			OfferingName:   "test-addon",
			OfferingFlavor: "test-flavor",
		},
	}

	// Test dependency discovery
	dependencies, err := options.discoverDependencies()
	assert.NoError(t, err)
	assert.Len(t, dependencies, 0)

	// Verify mock expectations were met
	mockService.AssertExpectations(t)
}
