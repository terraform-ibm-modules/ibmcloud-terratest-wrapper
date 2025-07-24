package testaddons

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"

	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

const defaultRegion = "us-south"
const defaultRegionYaml = "../common-dev-assets/common-go-assets/cloudinfo-region-vpc-gen2-prefs.yaml"
const ibmcloudApiKeyVar = "TF_VAR_ibmcloud_api_key"

type TestAddonOptions struct {
	// REQUIRED: a pointer to an initialized testing object.
	// Typically you would assign the test object used in the unit test.
	Testing *testing.T `copier:"-"`

	// The default constructors will use this map to check that all required environment variables are set properly.
	// If any are missing, the test will fail.
	RequiredEnvironmentVars map[string]string

	// Only required if using the WithVars constructor, as this value will then populate the `resource_group` input variable.
	// This resource group will be used to create the project
	ResourceGroup string

	// REQUIRED: the string prefix that will be prepended to all resource names, typically sent in as terraform input variable.
	// Set this value in the default constructors and a unique 6-digit random string will be appended.
	// Can then be referenced after construction and used as unique variable.
	//
	// Example:
	// Supplied to constructor = `my-test`
	// After constructor = `my-test-xu5oby`
	Prefix string

	ProjectName              string
	ProjectDescription       string
	ProjectLocation          string
	ProjectDestroyOnDelete   *bool
	ProjectMonitoringEnabled *bool
	ProjectAutoDeploy        *bool
	ProjectEnvironments      []project.EnvironmentPrototype

	CloudInfoService cloudinfo.CloudInfoServiceI // OPTIONAL: Supply if you need multiple tests to share info service and data

	// CatalogUseExisting If set to true, the test will use an existing catalog.
	CatalogUseExisting bool
	// CatalogName The name of the catalog to create and deploy to.
	CatalogName string

	// SharedCatalog If set to true (default), catalogs and offerings will be shared across tests using the same TestOptions object.
	// When false, each test will create its own catalog and offering, which is useful for isolation but less efficient.
	// This applies to both individual tests and matrix tests.
	SharedCatalog *bool

	// Internal Use
	// catalog the catalog instance in use.
	catalog *catalogmanagementv1.Catalog

	// internal use
	// offering the offering created in the catalog.
	offering *catalogmanagementv1.Offering

	// AddonConfig The configuration for the addon to deploy.
	AddonConfig cloudinfo.AddonConfig

	// DeployTimeoutMinutes The number of minutes to wait for the stack to deploy. Also used for undeploy. Default is 6 hours.
	DeployTimeoutMinutes int

	// If you want to skip teardown use this flag
	SkipTestTearDown  bool
	SkipUndeploy      bool
	SkipProjectDelete bool

	// SkipInfrastructureDeployment If set to true, the test will skip the infrastructure deployment and undeploy operations.
	// All other validations and setup will still be performed.
	SkipInfrastructureDeployment bool

	// SkipLocalChangeCheck If set to true, the test will not check for local changes before deploying.
	SkipLocalChangeCheck bool
	// SkipRefValidation If set to true, the test will not check for reference validation before deploying.
	SkipRefValidation bool
	// SkipDependencyValidatio If set to true, the test will not check for dependency validation before deploying
	SkipDependencyValidation bool

	// InputValidationRetries The number of retry attempts for input validation (default: 3)
	// This handles timing issues where the backend database hasn't been updated yet after configuration changes
	InputValidationRetries int
	// InputValidationRetryDelay The delay between retry attempts for input validation (default: 2 seconds)
	InputValidationRetryDelay time.Duration

	// VerboseValidationErrors If set to true, shows detailed individual error messages instead of consolidated summary
	VerboseValidationErrors bool
	// EnhancedTreeValidationOutput If set to true, shows dependency tree with validation status annotations
	EnhancedTreeValidationOutput bool
	// LocalChangesIgnorePattern List of regex patterns to ignore files or directories when checking for local changes.
	LocalChangesIgnorePattern []string

	// TestCaseName Optional custom identifier for log messages. When specified, log output will show:
	// "[TestFunction - ADDON - TestCaseName]" instead of using the project name.
	// Matrix tests automatically set this using the AddonTestCase.Name field.
	TestCaseName string

	// QuietMode If set to true, detailed logs are buffered and only shown on test failure.
	// When false, all logs are shown immediately. Default is false.
	QuietMode bool

	// VerboseOnFailure If set to true, detailed logs are shown when a test fails.
	// Only effective when QuietMode is true. Default is true.
	VerboseOnFailure bool

	// internal use
	currentProject       *project.Project
	currentProjectConfig *cloudinfo.ProjectsConfig
	deployedConfigs      *cloudinfo.DeployedAddonsDetails // Store deployed configs for validation

	currentBranch    *string
	currentBranchUrl *string

	// Hooks These allow us to inject custom code into the test process
	// example to set a hook:
	// options.PreDeployHook = func(options *TestProjectsOptions) error {
	//     // do something
	//     return nil
	// }
	PreDeployHook    func(options *TestAddonOptions) error // In upgrade tests, this hook will be called before the deploy
	PostDeployHook   func(options *TestAddonOptions) error // In upgrade tests, this hook will be called after the deploy
	PreUndeployHook  func(options *TestAddonOptions) error // If this fails, the undeploy will continue
	PostUndeployHook func(options *TestAddonOptions) error

	Logger common.Logger

	// PermutationTestReport stores results for permutation test reporting
	PermutationTestReport *PermutationTestReport
	// CollectResults enables collection of test results for final reporting
	CollectResults bool
	// Internal fields for error collection during test execution
	lastValidationResult *ValidationResult
	lastTransientErrors  []string
	lastRuntimeErrors    []string
}

// TestAddonsOptionsDefault Default constructor for TestAddonOptions
// This function will accept an existing instance of
// TestAddonOptions values, and return a new instance of TestAddonOptions with the original values set along with appropriate
// default values for any properties that were not set in the original options.
// Summary of default values:
// - Prefix: original prefix with a unique 6-digit random string appended
func TestAddonsOptionsDefault(originalOptions *TestAddonOptions) *TestAddonOptions {
	newOptions, err := originalOptions.Clone()
	require.NoError(originalOptions.Testing, err)

	// Handle empty prefix case to avoid leading hyphen
	if newOptions.Prefix == "" {
		newOptions.Prefix = common.UniqueId()
	} else {
		newOptions.Prefix = fmt.Sprintf("%s-%s", newOptions.Prefix, common.UniqueId())
	}
	newOptions.AddonConfig.Prefix = newOptions.Prefix

	// Verify required environment variables are set - better to do this now rather than retry and fail with every attempt
	// Only check if RequiredEnvironmentVars hasn't been explicitly set (for unit tests that don't need env vars)
	if newOptions.RequiredEnvironmentVars == nil {
		checkVariables := []string{ibmcloudApiKeyVar}
		newOptions.RequiredEnvironmentVars = common.GetRequiredEnvVars(newOptions.Testing, checkVariables)
	}

	if newOptions.CatalogName == "" {
		newOptions.CatalogName = fmt.Sprintf("addon-test-catalog-%s", newOptions.Prefix)
	}
	if newOptions.ProjectName == "" {
		newOptions.ProjectName = fmt.Sprintf("addon-%s", newOptions.Prefix)
	}
	if newOptions.ProjectDescription == "" {
		newOptions.ProjectDescription = fmt.Sprintf("Testing %s-addon", newOptions.Prefix)
	}

	if newOptions.ResourceGroup == "" {
		newOptions.ResourceGroup = "Default"
	}

	if newOptions.DeployTimeoutMinutes == 0 {
		newOptions.DeployTimeoutMinutes = 6 * 60
	}
	if newOptions.ProjectDestroyOnDelete == nil {
		newOptions.ProjectDestroyOnDelete = core.BoolPtr(true)
	}
	if newOptions.ProjectMonitoringEnabled == nil {
		newOptions.ProjectMonitoringEnabled = core.BoolPtr(true)
	}
	if newOptions.ProjectAutoDeploy == nil {
		newOptions.ProjectAutoDeploy = core.BoolPtr(true)
	}

	// We need to handle the bool default properly - default SharedCatalog to false for individual tests
	// Matrix tests will override this to true and handle cleanup automatically
	if newOptions.SharedCatalog == nil {
		newOptions.SharedCatalog = core.BoolPtr(false)
	}

	// Set default retry configuration for input validation
	if newOptions.InputValidationRetries <= 0 {
		newOptions.InputValidationRetries = 3
	}
	if newOptions.InputValidationRetryDelay <= 0 {
		newOptions.InputValidationRetryDelay = 2 * time.Second
	}

	// Always include default ignore patterns and append user patterns if provided
	defaultIgnorePatterns := []string{
		"^common-dev-assets$",   // Ignore submodule pointer changes for common-dev-assets
		"^common-dev-assets/.*", // Ignore changes in common-dev-assets directory
		"^tests/.*",             // Ignore changes in tests directory
		".*\\.json$",            // Ignore JSON files
		".*\\.out$",             // Ignore output files
	}

	if newOptions.LocalChangesIgnorePattern == nil {
		newOptions.LocalChangesIgnorePattern = defaultIgnorePatterns
	} else {
		// Append user patterns to default patterns
		newOptions.LocalChangesIgnorePattern = append(defaultIgnorePatterns, newOptions.LocalChangesIgnorePattern...)
	}

	// Set default logging behavior (VerboseOnFailure defaults to true)
	if !newOptions.VerboseOnFailure {
		newOptions.VerboseOnFailure = true
	}

	// Initialize logger if not already set to prevent nil pointer panics
	if newOptions.Logger == nil {
		testName := "addon-test"
		if newOptions.Testing != nil && newOptions.Testing.Name() != "" {
			testName = newOptions.Testing.Name()
		}

		// Use the QuietMode setting directly (defaults to false)
		newOptions.Logger = common.CreateSmartAutoBufferingLogger(testName, newOptions.QuietMode)
	}

	return newOptions
}

// Clone makes a deep copy of most fields on the Options object and returns it.
//
// NOTE: options.SshAgent and options.Logger CANNOT be deep copied (e.g., the SshAgent struct contains channels and
// listeners that can't be meaningfully copied), so the original values are retained.
func (options *TestAddonOptions) Clone() (*TestAddonOptions, error) {
	newOptions := &TestAddonOptions{}
	if err := copier.Copy(newOptions, options); err != nil {
		return nil, err
	}

	// the Copy library does not handle pointer of struct very well so we want to manually take care of our
	// pointers to other complex structs
	newOptions.Testing = options.Testing

	return newOptions, nil
}

// copy creates a deep copy of TestAddonOptions for use in matrix tests
// This allows BaseOptions to be safely shared across test cases
// copyBoolPointer creates a deep copy of a bool pointer
func copyBoolPointer(original *bool) *bool {
	if original == nil {
		return nil
	}
	copied := *original
	return &copied
}

func (options *TestAddonOptions) copy() *TestAddonOptions {
	if options == nil {
		return nil
	}

	copied := &TestAddonOptions{
		Testing:                      options.Testing, // Will be overridden per test case
		RequiredEnvironmentVars:      options.RequiredEnvironmentVars,
		ResourceGroup:                options.ResourceGroup,
		Prefix:                       options.Prefix,
		ProjectName:                  options.ProjectName,
		ProjectDescription:           options.ProjectDescription,
		ProjectLocation:              options.ProjectLocation,
		ProjectDestroyOnDelete:       options.ProjectDestroyOnDelete,
		ProjectMonitoringEnabled:     options.ProjectMonitoringEnabled,
		ProjectAutoDeploy:            options.ProjectAutoDeploy,
		ProjectEnvironments:          options.ProjectEnvironments,
		CloudInfoService:             options.CloudInfoService,
		CatalogUseExisting:           options.CatalogUseExisting,
		CatalogName:                  options.CatalogName,
		SharedCatalog:                copyBoolPointer(options.SharedCatalog),
		AddonConfig:                  options.AddonConfig, // Note: shallow copy, will be overridden
		DeployTimeoutMinutes:         options.DeployTimeoutMinutes,
		SkipTestTearDown:             options.SkipTestTearDown,
		SkipUndeploy:                 options.SkipUndeploy,
		SkipProjectDelete:            options.SkipProjectDelete,
		SkipInfrastructureDeployment: options.SkipInfrastructureDeployment,
		SkipLocalChangeCheck:         options.SkipLocalChangeCheck,
		SkipRefValidation:            options.SkipRefValidation,
		SkipDependencyValidation:     options.SkipDependencyValidation,
		VerboseValidationErrors:      options.VerboseValidationErrors,
		EnhancedTreeValidationOutput: options.EnhancedTreeValidationOutput,
		LocalChangesIgnorePattern:    options.LocalChangesIgnorePattern,
		TestCaseName:                 options.TestCaseName,
		InputValidationRetries:       options.InputValidationRetries,
		InputValidationRetryDelay:    options.InputValidationRetryDelay,
		PreDeployHook:                options.PreDeployHook,
		PostDeployHook:               options.PostDeployHook,
		PreUndeployHook:              options.PreUndeployHook,
		PostUndeployHook:             options.PostUndeployHook,
		Logger:                       options.Logger,
		QuietMode:                    options.QuietMode,

		// These fields are not copied as they are managed per test instance
		catalog:              nil,
		offering:             nil,
		currentProject:       nil,
		currentProjectConfig: nil,
		deployedConfigs:      nil,
		currentBranch:        nil,
		currentBranchUrl:     nil,
	}

	return copied
}

// CleanupSharedResources cleans up shared catalog and offering resources
// This method is useful for cleaning up shared catalogs when using SharedCatalog=true with individual tests.
// For matrix tests, cleanup happens automatically and you don't need to call this method.
//
// Example usage:
//
//	options := testaddons.TestAddonsOptionsDefault(&testaddons.TestAddonOptions{
//	    Testing: t,
//	    Prefix: "shared-test",
//	    ResourceGroup: "my-rg",
//	    SharedCatalog: core.BoolPtr(true),
//	})
//	defer options.CleanupSharedResources() // Ensure cleanup happens
//
//	// Run multiple tests that share the catalog
//	err1 := options.RunAddonTest()
//	err2 := options.RunAddonTest()
func (options *TestAddonOptions) CleanupSharedResources() {
	if options.catalog != nil {
		options.Logger.ShortInfo(fmt.Sprintf("Deleting the shared catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		err := options.CloudInfoService.DeleteCatalog(*options.catalog.ID)
		if err != nil {
			options.Logger.ShortError(fmt.Sprintf("Error deleting the shared catalog: %v", err))
		} else {
			options.Logger.ShortInfo(fmt.Sprintf("Deleted the shared catalog %s with ID %s", *options.catalog.Label, *options.catalog.ID))
		}
	}
}

// collectTestResult creates a PermutationTestResult from test execution
func (options *TestAddonOptions) collectTestResult(testName, testPrefix string, addonConfig cloudinfo.AddonConfig, testError error) PermutationTestResult {
	// Create base result with complete addon configuration including all dependencies
	// First entry is the main addon (always enabled), followed by all dependencies
	completeAddonConfig := append([]cloudinfo.AddonConfig{addonConfig}, addonConfig.Dependencies...)

	result := PermutationTestResult{
		Name:        testName,
		Prefix:      testPrefix,
		AddonConfig: completeAddonConfig,
		Passed:      testError == nil,
	}

	// Collect validation errors if available
	if options.lastValidationResult != nil {
		result.ValidationResult = options.lastValidationResult
	}

	// Collect other error categories (simplified)
	if options.lastTransientErrors != nil {
		result.TransientErrors = append(result.TransientErrors, options.lastTransientErrors...)
	}

	if options.lastRuntimeErrors != nil {
		result.RuntimeErrors = append(result.RuntimeErrors, options.lastRuntimeErrors...)
	}

	// If test failed, parse and categorize the main error
	if testError != nil {
		options.categorizeError(testError, &result)
	}

	// Reset error collection fields for next test
	options.lastValidationResult = nil
	options.lastTransientErrors = nil
	options.lastRuntimeErrors = nil

	return result
}

// categorizeError parses the main test error and categorizes it into one of three simplified categories
func (options *TestAddonOptions) categorizeError(testError error, result *PermutationTestResult) {
	errorStr := testError.Error()

	// Check if we already have detailed error info
	hasDetailedErrors := (result.ValidationResult != nil && !result.ValidationResult.IsValid) ||
		len(result.TransientErrors) > 0 || len(result.RuntimeErrors) > 0

	// If we don't have detailed errors, try to categorize the main error
	if !hasDetailedErrors {
		switch {
		// VALIDATION ERRORS: Configuration, dependency, and input validation issues
		case strings.Contains(errorStr, "missing required inputs"):
			options.addValidationError(result, errorStr, "missing_inputs")
		case strings.Contains(errorStr, "dependency validation failed"):
			options.addValidationError(result, errorStr, "dependency_validation")
		case strings.Contains(errorStr, "unexpected configs"):
			options.addValidationError(result, errorStr, "unexpected_configs")
		case strings.Contains(errorStr, "should not be deployed"):
			options.addValidationError(result, errorStr, "unexpected_deployment")
		case strings.Contains(errorStr, "configuration validation"):
			options.addValidationError(result, errorStr, "configuration")

		// TRANSIENT ERRORS: API failures, timeouts, infrastructure issues
		case strings.Contains(errorStr, "deployment timeout") || strings.Contains(errorStr, "TriggerDeployAndWait"):
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case strings.Contains(errorStr, "TriggerUnDeployAndWait"):
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case strings.Contains(errorStr, "5") && strings.Contains(errorStr, " error"): // 5xx errors
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case strings.Contains(errorStr, "timeout"):
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case strings.Contains(errorStr, "rate limit"):
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case strings.Contains(errorStr, "network") || strings.Contains(errorStr, "connection"):
			result.TransientErrors = append(result.TransientErrors, errorStr)

		// RUNTIME ERRORS: Go panics, nil pointers, code bugs
		case strings.Contains(errorStr, "panic:") || strings.Contains(errorStr, "runtime error"):
			result.RuntimeErrors = append(result.RuntimeErrors, errorStr)
		case strings.Contains(errorStr, "nil pointer"):
			result.RuntimeErrors = append(result.RuntimeErrors, errorStr)

		default:
			// Default to transient error for unknown issues (likely infrastructure)
			result.TransientErrors = append(result.TransientErrors, errorStr)
		}
	} else {
		// We have detailed errors, but still add the main error if it's not redundant
		if !strings.Contains(errorStr, "Addon Test had an unexpected error") {
			// Categorize the main error even when we have detailed errors
			options.categorizeError(testError, &PermutationTestResult{}) // Recursive call to get category
		}
	}
}

// addValidationError helper function to add validation errors to ValidationResult
func (options *TestAddonOptions) addValidationError(result *PermutationTestResult, errorStr string, errorType string) {
	if result.ValidationResult == nil {
		result.ValidationResult = &ValidationResult{
			IsValid:             false,
			Messages:            []string{},
			MissingInputs:       []string{},
			ConfigurationErrors: []string{},
		}
	}

	switch errorType {
	case "missing_inputs":
		result.ValidationResult.MissingInputs = append(result.ValidationResult.MissingInputs, errorStr)
	case "configuration":
		result.ValidationResult.ConfigurationErrors = append(result.ValidationResult.ConfigurationErrors, errorStr)
	default:
		// Parse detailed validation info or add to messages
		validationResult := options.parseValidationError(errorStr)
		if validationResult != nil {
			// Merge parsed validation result
			options.mergeValidationResults(result.ValidationResult, validationResult)
		} else {
			result.ValidationResult.Messages = append(result.ValidationResult.Messages, errorStr)
		}
	}
}

// mergeValidationResults merges two ValidationResult objects
func (options *TestAddonOptions) mergeValidationResults(target *ValidationResult, source *ValidationResult) {
	target.DependencyErrors = append(target.DependencyErrors, source.DependencyErrors...)
	target.UnexpectedConfigs = append(target.UnexpectedConfigs, source.UnexpectedConfigs...)
	target.MissingConfigs = append(target.MissingConfigs, source.MissingConfigs...)
	target.MissingInputs = append(target.MissingInputs, source.MissingInputs...)
	target.ConfigurationErrors = append(target.ConfigurationErrors, source.ConfigurationErrors...)
	target.Messages = append(target.Messages, source.Messages...)
	if !source.IsValid {
		target.IsValid = false
	}
}

// parseValidationError parses validation error messages and creates detailed ValidationResult objects
func (options *TestAddonOptions) parseValidationError(errorStr string) *ValidationResult {
	validationResult := &ValidationResult{
		IsValid:           false,
		DependencyErrors:  []cloudinfo.DependencyError{},
		UnexpectedConfigs: []cloudinfo.OfferingReferenceDetail{},
		MissingConfigs:    []cloudinfo.OfferingReferenceDetail{},
		Messages:          []string{},
	}

	// Parse "dependency validation failed: X unexpected configs" pattern
	if strings.Contains(errorStr, "dependency validation failed:") && strings.Contains(errorStr, "unexpected configs") {
		// Extract the number of unexpected configs
		parts := strings.Split(errorStr, ":")
		if len(parts) >= 2 {
			configInfo := strings.TrimSpace(parts[1])
			validationResult.Messages = append(validationResult.Messages, configInfo)

			// Try to extract specific unexpected config names if available
			// This would need more detailed parsing based on actual error format
			// For now, add the general message
			return validationResult
		}
	}

	// Parse "Input validation failed after dependency validation" pattern
	// This usually indicates missing required inputs due to disabled dependencies
	if strings.Contains(errorStr, "Input validation failed after dependency validation") {
		validationResult.Messages = append(validationResult.Messages, "Input validation failed after dependency validation")
		return validationResult
	}

	// Parse specific config names from error messages like:
	// "deploy-arch-ibm-cloud-logs (v1.5.6, fully-configurable) - should not be deployed"
	if strings.Contains(errorStr, "should not be deployed") {
		// Extract config details
		configName := extractConfigNameFromError(errorStr)
		version := extractVersionFromError(errorStr)
		flavor := extractFlavorFromError(errorStr)

		if configName != "" {
			unexpectedConfig := cloudinfo.OfferingReferenceDetail{
				Name:    configName,
				Version: version,
			}

			// Add flavor information if available
			if flavor != "" {
				unexpectedConfig.Flavor = cloudinfo.Flavor{Name: flavor}
			}

			validationResult.UnexpectedConfigs = append(validationResult.UnexpectedConfigs, unexpectedConfig)
			return validationResult
		}
	}

	// Parse missing dependency patterns
	if strings.Contains(errorStr, "missing:") && strings.Contains(errorStr, "(missing:") {
		// This indicates missing required inputs, which is a validation issue
		validationResult.Messages = append(validationResult.Messages, errorStr)
		return validationResult
	}

	// If we couldn't parse specific details, return nil to use fallback
	return nil
}

// Helper functions to extract config details from error messages
func extractConfigNameFromError(errorStr string) string {
	// Look for patterns like "deploy-arch-ibm-cloud-logs (v1.5.6, fully-configurable)"
	if idx := strings.Index(errorStr, " (v"); idx != -1 {
		return strings.TrimSpace(errorStr[:idx])
	}

	// Look for patterns with just config name before " - should not be deployed"
	if idx := strings.Index(errorStr, " - should not be deployed"); idx != -1 {
		return strings.TrimSpace(errorStr[:idx])
	}

	return ""
}

func extractVersionFromError(errorStr string) string {
	// Look for version pattern like "(v1.5.6"
	if start := strings.Index(errorStr, "(v"); start != -1 {
		start += 2 // Skip "(v"
		if end := strings.Index(errorStr[start:], ","); end != -1 {
			return strings.TrimSpace(errorStr[start : start+end])
		}
		if end := strings.Index(errorStr[start:], ")"); end != -1 {
			return strings.TrimSpace(errorStr[start : start+end])
		}
	}
	return ""
}

func extractFlavorFromError(errorStr string) string {
	// Look for flavor pattern like ", fully-configurable)"
	if start := strings.Index(errorStr, ", "); start != -1 {
		start += 2 // Skip ", "
		if end := strings.Index(errorStr[start:], ")"); end != -1 {
			return strings.TrimSpace(errorStr[start : start+end])
		}
	}
	return ""
}
