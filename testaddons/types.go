package testaddons

import (
	"time"

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
	// BaseOptions contains common options that apply to all test cases (required)
	// Reduces boilerplate by providing shared configuration across all test cases
	BaseOptions *TestAddonOptions
	// BaseSetupFunc is called to customize TestAddonOptions for each test case (optional)
	// Receives a copy of BaseOptions to customize for the specific test case
	BaseSetupFunc func(baseOptions *TestAddonOptions, testCase AddonTestCase) *TestAddonOptions
	// AddonConfigFunc is called to create the addon configuration for each test case
	AddonConfigFunc func(options *TestAddonOptions, testCase AddonTestCase) cloudinfo.AddonConfig
	// StaggerDelay is the time delay between starting each parallel test (optional)
	// This helps prevent rate limiting by spacing out API calls across parallel tests.
	// Default is applied if not specified. Set to 0 to disable staggering.
	// Recommended values: 2-15 seconds for most scenarios, 20-30 seconds for high-volume tests.
	StaggerDelay *time.Duration
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

// Helper functions for common stagger delay configurations

// StaggerDelay creates a stagger delay with the specified duration
// Use this to customize the delay between parallel test starts to prevent rate limiting
func StaggerDelay(delay time.Duration) *time.Duration {
	return &delay
}
