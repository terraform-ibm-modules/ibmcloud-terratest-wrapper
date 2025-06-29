package testaddons

import (
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// AddonTestCase defines the structure for addon test cases used in parallel matrix testing
type AddonTestCase struct {
	// Name is the test case name that will appear in test output
	Name string
	// Prefix is the unique prefix for resource naming in this test case
	Prefix string
	// Dependencies are the addon dependencies to configure for this test case
	Dependencies []cloudinfo.AddonConfig
	// Inputs are additional inputs to merge with the base addon configuration
	Inputs map[string]interface{}
	// SkipTearDown can be set to true to skip cleanup for this specific test case
	SkipTearDown bool
	// SkipInfrastructureDeployment can be set to true to skip infrastructure deployment and undeploy operations for this specific test case
	SkipInfrastructureDeployment bool
}

// AddonTestMatrix provides a convenient way to run multiple addon test cases in parallel
type AddonTestMatrix struct {
	// TestCases are the individual test cases to run
	TestCases []AddonTestCase
	// BaseOptions contains common options that apply to all test cases (optional)
	// When provided, reduces boilerplate by avoiding repetition in BaseSetupFunc
	BaseOptions *TestAddonOptions
	// BaseSetupFunc is called to create or customize TestAddonOptions for each test case
	// If BaseOptions is provided, this function receives a copy of BaseOptions to customize
	// If BaseOptions is nil, this function must create the options from scratch (legacy behavior)
	BaseSetupFunc func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions
	// AddonConfigFunc is called to create the addon configuration for each test case
	AddonConfigFunc func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig
}

// BuildActuallyDeployedResult contains the results of building the actually deployed list
type BuildActuallyDeployedResult struct {
	ActuallyDeployedList []cloudinfo.OfferingReferenceDetail
	Warnings             []string
	Errors               []string
}

// ValidationResult contains the results of dependency validation
type ValidationResult struct {
	IsValid           bool
	DependencyErrors  []cloudinfo.DependencyError
	UnexpectedConfigs []cloudinfo.OfferingReferenceDetail
	MissingConfigs    []cloudinfo.OfferingReferenceDetail
	Messages          []string
}

// DependencyGraphResult contains the results of building a dependency graph
type DependencyGraphResult struct {
	Graph                map[string][]cloudinfo.OfferingReferenceDetail // Using string key for offering identity
	ExpectedDeployedList []cloudinfo.OfferingReferenceDetail
	Visited              map[string]bool
}
