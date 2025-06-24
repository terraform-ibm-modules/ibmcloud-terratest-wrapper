package testaddons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// TestAddonTestCaseStructure tests that the new AddonTestCase structure is properly defined
func TestAddonTestCaseStructure(t *testing.T) {
	// Test creating an AddonTestCase with all fields
	testCase := AddonTestCase{
		Name:   "TestCase1",
		Prefix: "test-prefix",
		Dependencies: []cloudinfo.AddonConfig{
			{
				OfferingName:   "test-offering",
				OfferingFlavor: "test-flavor",
			},
		},
		Inputs: map[string]interface{}{
			"test-input": "test-value",
		},
		SkipTearDown:                 true,
		SkipInfrastructureDeployment: true,
	}

	// Verify the structure is properly initialized
	assert.Equal(t, "TestCase1", testCase.Name)
	assert.Equal(t, "test-prefix", testCase.Prefix)
	assert.Len(t, testCase.Dependencies, 1)
	assert.Equal(t, "test-offering", testCase.Dependencies[0].OfferingName)
	assert.Equal(t, "test-flavor", testCase.Dependencies[0].OfferingFlavor)
	assert.Equal(t, "test-value", testCase.Inputs["test-input"])
	assert.True(t, testCase.SkipTearDown)
	assert.True(t, testCase.SkipInfrastructureDeployment)
}

// TestAddonTestMatrix tests that the AddonTestMatrix structure is properly defined
func TestAddonTestMatrix(t *testing.T) {
	t.Run("WithoutBaseOptions", func(t *testing.T) {
		matrix := AddonTestMatrix{
			TestCases: []AddonTestCase{
				{Name: "Case1", Prefix: "prefix1"},
				{Name: "Case2", Prefix: "prefix2"},
			},
			BaseSetupFunc: func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions {
				// When no BaseOptions provided, baseOptions will be nil
				assert.Nil(t, baseOptions)
				return &TestAddonOptions{
					Prefix: testCase.Prefix,
				}
			},
			AddonConfigFunc: func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig {
				return cloudinfo.AddonConfig{
					Prefix:         options.Prefix,
					OfferingName:   "test-addon",
					OfferingFlavor: "test-flavor",
				}
			},
		}

		// Verify structure
		assert.Len(t, matrix.TestCases, 2)
		assert.Equal(t, "Case1", matrix.TestCases[0].Name)
		assert.NotNil(t, matrix.BaseSetupFunc)
		assert.NotNil(t, matrix.AddonConfigFunc)

		// Test function calls
		options := matrix.BaseSetupFunc(nil, matrix.TestCases[0])
		assert.Equal(t, "prefix1", options.Prefix)

		config := matrix.AddonConfigFunc(options, matrix.TestCases[0])
		assert.Equal(t, "test-addon", config.OfferingName)
	})

	t.Run("WithBaseOptions", func(t *testing.T) {
		baseOptions := &TestAddonOptions{
			Prefix:        "base-prefix",
			ResourceGroup: "base-rg",
			SharedCatalog: &[]bool{true}[0],
		}

		matrix := AddonTestMatrix{
			BaseOptions: baseOptions,
			TestCases:   []AddonTestCase{{Name: "TestCase1", Prefix: "override-prefix"}},
			BaseSetupFunc: func(baseOpts *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions {
				assert.NotNil(t, baseOpts)
				assert.NotSame(t, baseOptions, baseOpts) // Should be a copy
				baseOpts.Prefix = testCase.Prefix
				return baseOpts
			},
			AddonConfigFunc: func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig {
				return cloudinfo.AddonConfig{
					Prefix:         options.Prefix,
					OfferingName:   "test-addon",
					OfferingFlavor: "test-flavor",
				}
			},
		}

		// Verify BaseOptions
		assert.NotNil(t, matrix.BaseOptions)
		assert.Equal(t, "base-prefix", matrix.BaseOptions.Prefix)
		assert.True(t, *matrix.BaseOptions.SharedCatalog)

		// Test function with BaseOptions (should receive a copy)
		copiedOptions := baseOptions.copy()
		testOptions := matrix.BaseSetupFunc(copiedOptions, matrix.TestCases[0])
		assert.Equal(t, "override-prefix", testOptions.Prefix)
		assert.Equal(t, "base-rg", testOptions.ResourceGroup)
	})
}

// TestTestAddonOptionsCopy tests the copy() method
func TestTestAddonOptionsCopy(t *testing.T) {
	t.Run("FullCopy", func(t *testing.T) {
		original := &TestAddonOptions{
			Prefix:               "test-prefix",
			ResourceGroup:        "test-rg",
			SharedCatalog:        &[]bool{true}[0],
			TestCaseName:         "original-test",
			DeployTimeoutMinutes: 120,
		}

		copied := original.copy()

		// Verify all fields are copied
		assert.Equal(t, original.Prefix, copied.Prefix)
		assert.Equal(t, original.ResourceGroup, copied.ResourceGroup)
		assert.Equal(t, *original.SharedCatalog, *copied.SharedCatalog)
		assert.Equal(t, original.TestCaseName, copied.TestCaseName)
		assert.Equal(t, original.DeployTimeoutMinutes, copied.DeployTimeoutMinutes)

		// Verify it's a deep copy
		copied.Prefix = "modified-prefix"
		*copied.SharedCatalog = false
		assert.NotEqual(t, original.Prefix, copied.Prefix)
		assert.True(t, *original.SharedCatalog) // Original unchanged
	})

	t.Run("NilCopy", func(t *testing.T) {
		var original *TestAddonOptions
		copied := original.copy()
		assert.Nil(t, copied)
	})
}

// TestSharedCatalogAndTeardown tests SharedCatalog behavior and demonstrates teardown patterns
func TestSharedCatalogAndTeardown(t *testing.T) {
	// Test basic SharedCatalog option behavior
	t.Run("OptionBehavior", func(t *testing.T) {
		// Default behavior
		options1 := &TestAddonOptions{}
		assert.Nil(t, options1.SharedCatalog)

		// Explicit values
		options2 := &TestAddonOptions{SharedCatalog: &[]bool{true}[0]}
		assert.True(t, *options2.SharedCatalog)

		options3 := &TestAddonOptions{SharedCatalog: &[]bool{false}[0]}
		assert.False(t, *options3.SharedCatalog)
	})

	// Test teardown scenarios with examples
	teardownScenarios := []struct {
		name           string
		sharedCatalog  *bool
		isMatrixTest   bool
		shouldCleanup  bool
		cleanupPattern string
	}{
		{
			name:           "IndividualTestDefault",
			sharedCatalog:  nil,
			isMatrixTest:   false,
			shouldCleanup:  true,
			cleanupPattern: "Automatic cleanup in test teardown",
		},
		{
			name:           "IndividualTestPrivate",
			sharedCatalog:  &[]bool{false}[0],
			isMatrixTest:   false,
			shouldCleanup:  true,
			cleanupPattern: "Automatic cleanup in test teardown",
		},
		{
			name:           "IndividualTestShared",
			sharedCatalog:  &[]bool{true}[0],
			isMatrixTest:   false,
			shouldCleanup:  false,
			cleanupPattern: "Manual cleanup: testaddons.CleanupSharedResources(t, options)",
		},
		{
			name:           "MatrixTest",
			sharedCatalog:  &[]bool{false}[0], // Matrix overrides this
			isMatrixTest:   true,
			shouldCleanup:  false,
			cleanupPattern: "Central cleanup by matrix framework",
		},
	}

	for _, scenario := range teardownScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Test teardown logic
			var shouldCleanup bool
			if scenario.isMatrixTest {
				shouldCleanup = false // Matrix tests never cleanup individually
			} else {
				shouldCleanup = scenario.sharedCatalog == nil || !*scenario.sharedCatalog
			}

			assert.Equal(t, scenario.shouldCleanup, shouldCleanup)
			t.Logf("Test: %s - Cleanup pattern: %s", scenario.name, scenario.cleanupPattern)
		})
	}

	// Example of manual cleanup pattern
	t.Run("ManualCleanupExample", func(t *testing.T) {
		sharedTests := []*TestAddonOptions{
			{Prefix: "test1", SharedCatalog: &[]bool{true}[0]},
			{Prefix: "test2", SharedCatalog: &[]bool{true}[0]},
		}

		for _, testOpts := range sharedTests {
			if testOpts.SharedCatalog != nil && *testOpts.SharedCatalog {
				// Example: testaddons.CleanupSharedResources(t, testOpts)
				t.Logf("Would cleanup shared resources for test '%s'", testOpts.Prefix)
			}
		}
		assert.Len(t, sharedTests, 2, "Example demonstrates cleanup for multiple shared tests")
	})
}

// TestMatrixConfigurationFeatures tests input merging and dependency handling
func TestMatrixConfigurationFeatures(t *testing.T) {
	t.Run("InputMerging", func(t *testing.T) {
		baseOptions := &TestAddonOptions{Testing: t, Prefix: "input-test"}
		testCase := AddonTestCase{
			Name:   "InputTest",
			Prefix: "input",
			Inputs: map[string]interface{}{
				"region":      "eu-gb",
				"environment": "test",
			},
		}

		matrix := AddonTestMatrix{
			TestCases:   []AddonTestCase{testCase},
			BaseOptions: baseOptions,
			AddonConfigFunc: func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig {
				return cloudinfo.AddonConfig{
					Prefix: options.Prefix,
					Inputs: map[string]interface{}{
						"prefix": options.Prefix,
						"region": "us-south", // Default, should be overridden
					},
				}
			},
		}

		// Simulate input merging
		testOptions := baseOptions.copy()
		testOptions.Prefix = testCase.Prefix
		config := matrix.AddonConfigFunc(testOptions, testCase)

		// Merge test case inputs
		if testCase.Inputs != nil {
			if config.Inputs == nil {
				config.Inputs = make(map[string]interface{})
			}
			for key, value := range testCase.Inputs {
				config.Inputs[key] = value
			}
		}

		// Verify merging
		assert.Equal(t, "input", config.Prefix)
		assert.Equal(t, "eu-gb", config.Inputs["region"], "Test case input should override default")
		assert.Equal(t, "test", config.Inputs["environment"], "Test case input should be added")
		assert.Equal(t, "input", config.Inputs["prefix"], "Base input should be preserved")
	})

	t.Run("DependencyHandling", func(t *testing.T) {
		dependency := cloudinfo.AddonConfig{
			OfferingName:   "dependency-addon",
			OfferingFlavor: "minimal",
			Enabled:        &[]bool{true}[0],
		}

		testCase := AddonTestCase{
			Name:         "DependencyTest",
			Prefix:       "dep",
			Dependencies: []cloudinfo.AddonConfig{dependency},
		}

		matrix := AddonTestMatrix{
			TestCases: []AddonTestCase{testCase},
			AddonConfigFunc: func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig {
				config := cloudinfo.AddonConfig{
					OfferingName: "main-addon",
					Prefix:       options.Prefix,
				}

				// Apply dependencies from test case
				if testCase.Dependencies != nil {
					config.Dependencies = testCase.Dependencies
				}

				return config
			},
		}

		// Test dependency handling
		config := matrix.AddonConfigFunc(&TestAddonOptions{Prefix: "dep"}, testCase)

		// Verify dependencies
		assert.Len(t, config.Dependencies, 1)
		assert.Equal(t, "dependency-addon", config.Dependencies[0].OfferingName)
		assert.Equal(t, "minimal", config.Dependencies[0].OfferingFlavor)
		assert.True(t, *config.Dependencies[0].Enabled)
	})
}
