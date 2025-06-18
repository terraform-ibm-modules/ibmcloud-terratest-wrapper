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

// TestAddonTestMatrix tests that the new AddonTestMatrix structure is properly defined
func TestAddonTestMatrix(t *testing.T) {
	// Test creating an AddonTestMatrix
	matrix := AddonTestMatrix{
		TestCases: []AddonTestCase{
			{Name: "Case1", Prefix: "prefix1"},
			{Name: "Case2", Prefix: "prefix2"},
		},
		BaseSetupFunc: func(testCase AddonTestCase) *TestAddonOptions {
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

	// Verify the structure is properly initialized
	assert.Len(t, matrix.TestCases, 2)
	assert.Equal(t, "Case1", matrix.TestCases[0].Name)
	assert.Equal(t, "prefix1", matrix.TestCases[0].Prefix)
	assert.NotNil(t, matrix.BaseSetupFunc)
	assert.NotNil(t, matrix.AddonConfigFunc)

	// Test that the functions work
	options := matrix.BaseSetupFunc(matrix.TestCases[0])
	assert.Equal(t, "prefix1", options.Prefix)

	config := matrix.AddonConfigFunc(options, matrix.TestCases[0])
	assert.Equal(t, "prefix1", config.Prefix)
	assert.Equal(t, "test-addon", config.OfferingName)
	assert.Equal(t, "test-flavor", config.OfferingFlavor)
}
