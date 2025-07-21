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
	// StaggerDelay is the time delay between starting each batch of parallel tests (optional)
	// This helps prevent rate limiting by spacing out API calls across parallel tests.
	// Default is 10 seconds if not specified. Set to 0 to disable staggering.
	// When using batched staggering (default), this controls the delay between batches.
	// Recommended values: 5-15 seconds for most scenarios, 20-30 seconds for high API sensitivity.
	StaggerDelay *time.Duration
	// StaggerBatchSize is the number of tests per batch for staggered execution (optional)
	// Tests are grouped into batches, with larger delays between batches and smaller delays within batches.
	// This prevents excessive delays for large test suites while maintaining API rate limiting protection.
	// Default is 8 tests per batch. Set to 0 to use linear staggering (original behavior).
	StaggerBatchSize *int
	// WithinBatchDelay is the delay between tests within the same batch (optional)
	// This provides fine-grained control over API call spacing within each batch.
	// Default is 2 seconds. Only used when StaggerBatchSize > 0.
	WithinBatchDelay *time.Duration
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
//
// Example usage patterns:
//
//  // Default batched staggering (8 tests per batch, 10s between batches, 2s within batches)
//  matrix := AddonTestMatrix{
//      TestCases: testCases,
//      BaseOptions: options,
//  }
//
//  // Custom batching for high-volume tests (20 tests per batch, 15s between batches, 1s within batches)
//  matrix := AddonTestMatrix{
//      TestCases: testCases,
//      BaseOptions: options,
//      StaggerDelay: StaggerDelay(15 * time.Second),
//      StaggerBatchSize: StaggerBatchSize(20),
//      WithinBatchDelay: WithinBatchDelay(1 * time.Second),
//  }
//
//  // Linear staggering (original behavior, not recommended for >20 tests)
//  matrix := AddonTestMatrix{
//      TestCases: testCases,
//      BaseOptions: options,
//      StaggerDelay: StaggerDelay(10 * time.Second),
//      StaggerBatchSize: StaggerBatchSize(0), // Disable batching
//  }

// StaggerDelay creates a stagger delay with the specified duration
// Use this to customize the delay between parallel test starts to prevent rate limiting
func StaggerDelay(delay time.Duration) *time.Duration {
	return &delay
}

// StaggerBatchSize creates a batch size configuration for staggered execution
// Use this to group tests into batches with smaller delays within batches and larger delays between batches
//
// Recommended values:
//   - 8-12: Default range, good for most scenarios
//   - 4-6: High API sensitivity environments
//   - 15-25: Low API sensitivity, faster execution
//   - 0: Disable batching (use linear staggering)
func StaggerBatchSize(size int) *int {
	return &size
}

// WithinBatchDelay creates a delay configuration for tests within the same batch
// Use this to fine-tune API call spacing within each batch of tests
//
// Recommended values:
//   - 1-3 seconds: Most scenarios
//   - 5+ seconds: High API sensitivity environments
//   - 0.5-1 second: Low API sensitivity, faster execution
func WithinBatchDelay(delay time.Duration) *time.Duration {
	return &delay
}

// PermutationTestResult contains the results of a single permutation test case
type PermutationTestResult struct {
	// Name is the test case name
	Name string
	// Prefix is the unique resource prefix used
	Prefix string
	// AddonConfig shows which addons were enabled/disabled
	AddonConfig []cloudinfo.AddonConfig
	// Passed indicates if the test passed
	Passed bool
	// ValidationResult contains dependency validation errors if any
	ValidationResult *ValidationResult
	// DeploymentErrors contains errors from TriggerDeployAndWait
	DeploymentErrors []error
	// UndeploymentErrors contains errors from TriggerUnDeployAndWait
	UndeploymentErrors []error
	// ConfigurationErrors contains setup and configuration errors
	ConfigurationErrors []string
	// RuntimeErrors contains panic and other runtime errors
	RuntimeErrors []string
	// MissingInputs contains list of missing required inputs
	MissingInputs []string
}

// PermutationTestReport contains the complete report for all permutation tests
type PermutationTestReport struct {
	// TotalTests is the total number of tests run
	TotalTests int
	// PassedTests is the number of tests that passed
	PassedTests int
	// FailedTests is the number of tests that failed
	FailedTests int
	// Results contains all individual test results
	Results []PermutationTestResult
	// StartTime is when the test suite started
	StartTime time.Time
	// EndTime is when the test suite completed
	EndTime time.Time
}
