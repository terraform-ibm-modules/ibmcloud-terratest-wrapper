package testaddons

import (
	"fmt"
	"regexp"
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

// ErrorType defines the category of errors for structured classification
type ErrorType int

const (
	ValidationError ErrorType = iota // Configuration, dependency, and input validation issues
	TransientError                   // API failures, timeouts, infrastructure issues
	RuntimeError                     // Go panics, nil pointers, code bugs
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ValidationError:
		return "ValidationError"
	case TransientError:
		return "TransientError"
	case RuntimeError:
		return "RuntimeError"
	default:
		return "UnknownError"
	}
}

// ErrorPattern defines a structured pattern for error classification
// replacing fragile strings.Contains() checks with regex patterns and confidence scores
type ErrorPattern struct {
	Pattern    *regexp.Regexp // Regex pattern to match error messages
	Type       ErrorType      // Category of error (validation, transient, runtime)
	Subtype    string         // Specific subtype for detailed categorization
	Confidence float64        // Confidence score (0.0 to 1.0) for classification accuracy
}

// errorPatterns defines the comprehensive set of error patterns for classification
// replacing the hardcoded switch statement with structured, maintainable patterns
var errorPatterns = []ErrorPattern{
	// VALIDATION ERRORS: Configuration, dependency, and input validation issues
	{regexp.MustCompile(`missing required inputs`), ValidationError, "missing_inputs", 0.95},
	{regexp.MustCompile(`dependency validation failed`), ValidationError, "dependency_validation", 0.90},
	{regexp.MustCompile(`unexpected configs`), ValidationError, "unexpected_configs", 0.90},
	{regexp.MustCompile(`should not be deployed`), ValidationError, "unexpected_deployment", 0.85},
	{regexp.MustCompile(`configuration validation`), ValidationError, "configuration", 0.80},

	// TRANSIENT ERRORS: API failures, timeouts, infrastructure issues
	{regexp.MustCompile(`deployment timeout|TriggerDeployAndWait`), TransientError, "deployment_timeout", 0.95},
	{regexp.MustCompile(`TriggerUnDeployAndWait`), TransientError, "undeploy_timeout", 0.95},
	{regexp.MustCompile(`5\d{2}.*error`), TransientError, "server_error", 0.90}, // 5xx errors
	{regexp.MustCompile(`timeout`), TransientError, "general_timeout", 0.80},
	{regexp.MustCompile(`rate limit`), TransientError, "rate_limit", 0.90},
	{regexp.MustCompile(`network|connection`), TransientError, "network_error", 0.85},

	// RUNTIME ERRORS: Go panics, nil pointers, code bugs
	{regexp.MustCompile(`panic:|runtime error`), RuntimeError, "panic", 0.95},
	{regexp.MustCompile(`nil pointer`), RuntimeError, "nil_pointer", 0.95},
}

// classifyError categorizes an error using structured patterns instead of hardcoded string matching
// Returns the best matching pattern with highest confidence score
func classifyError(errorStr string) (ErrorPattern, bool) {
	var bestMatch ErrorPattern
	var found bool
	highestConfidence := 0.0

	for _, pattern := range errorPatterns {
		if pattern.Pattern.MatchString(errorStr) {
			if pattern.Confidence > highestConfidence {
				bestMatch = pattern
				highestConfidence = pattern.Confidence
				found = true
			}
		}
	}

	return bestMatch, found
}

// ReferenceErrorDetector provides structured detection of reference-related errors
// replacing complex multi-condition string matching with reusable pattern detection
type ReferenceErrorDetector struct {
	RequiredPhrases []string // All phrases that must be present
	OptionalPhrases []string // Phrases that may be present (for future extensibility)
	ExcludePhrases  []string // Phrases that disqualify the match
}

// IsReferenceError determines if a message represents a member deployment reference error
// that should be treated as a warning rather than a failure
func (d *ReferenceErrorDetector) IsReferenceError(message string) bool {
	// All required phrases must be present
	for _, phrase := range d.RequiredPhrases {
		if !strings.Contains(message, phrase) {
			return false
		}
	}

	// No excluded phrases should be present
	for _, phrase := range d.ExcludePhrases {
		if strings.Contains(message, phrase) {
			return false
		}
	}

	return true
}

// memberDeploymentReferenceDetector is the configured detector for member deployment references
var memberDeploymentReferenceDetector = &ReferenceErrorDetector{
	RequiredPhrases: []string{
		"project reference requires",
		"member configuration",
		"to be deployed",
	},
	OptionalPhrases: []string{
		"Please deploy the referring configuration",
	},
	ExcludePhrases: []string{
		// Add any phrases that would disqualify this as a member deployment reference
	},
}

// IsMemberDeploymentReference uses structured detection instead of hardcoded boolean logic
// to determine if a reference error is about member deployment requirements
func IsMemberDeploymentReference(message string) bool {
	return memberDeploymentReferenceDetector.IsReferenceError(message)
}

// APIErrorType defines different types of API errors for structured classification
type APIErrorType int

const (
	APIKeyError          APIErrorType = iota // API key validation failures
	ProjectNotFoundError                     // Project not found (404) errors
	IntermittentError                        // Known transient service issues
	UnknownAPIError                          // Other API errors
)

// String returns the string representation of APIErrorType
func (aet APIErrorType) String() string {
	switch aet {
	case APIKeyError:
		return "APIKeyError"
	case ProjectNotFoundError:
		return "ProjectNotFoundError"
	case IntermittentError:
		return "IntermittentError"
	case UnknownAPIError:
		return "UnknownAPIError"
	default:
		return "UnknownAPIError"
	}
}

// APIErrorDetector provides structured detection of API-related errors
// replacing complex multi-condition string matching for error classification
type APIErrorDetector struct {
	ErrorType       APIErrorType
	RequiredPhrases []string
	StatusCodes     []string // HTTP status codes to match
}

// IsAPIError determines if an error message matches this detector's pattern
func (d *APIErrorDetector) IsAPIError(errorMessage string) bool {
	// All required phrases must be present
	for _, phrase := range d.RequiredPhrases {
		if !strings.Contains(errorMessage, phrase) {
			return false
		}
	}

	// At least one status code must be present (if specified)
	if len(d.StatusCodes) > 0 {
		statusMatched := false
		for _, code := range d.StatusCodes {
			if strings.Contains(errorMessage, code) {
				statusMatched = true
				break
			}
		}
		if !statusMatched {
			return false
		}
	}

	return true
}

// apiErrorDetectors is the configured set of API error detectors
var apiErrorDetectors = []*APIErrorDetector{
	{
		ErrorType:       APIKeyError,
		RequiredPhrases: []string{"Failed to validate api key token"},
		StatusCodes:     []string{"500"},
	},
	{
		ErrorType:       ProjectNotFoundError,
		RequiredPhrases: []string{"could not be found"},
		StatusCodes:     []string{"404"},
	},
	{
		ErrorType:       IntermittentError,
		RequiredPhrases: []string{"This is a known intermittent issue"},
		StatusCodes:     []string{}, // No specific status code required
	},
	{
		ErrorType:       IntermittentError,
		RequiredPhrases: []string{"known transient issue"},
		StatusCodes:     []string{}, // No specific status code required
	},
	{
		ErrorType:       IntermittentError,
		RequiredPhrases: []string{"typically transient"},
		StatusCodes:     []string{}, // No specific status code required
	},
}

// ClassifyAPIError uses structured detection to categorize API errors
// replacing fragile multi-condition string matching
func ClassifyAPIError(errorMessage string) (APIErrorType, bool) {
	for _, detector := range apiErrorDetectors {
		if detector.IsAPIError(errorMessage) {
			return detector.ErrorType, true
		}
	}
	return UnknownAPIError, false
}

// IsSkippableAPIError determines if an API error should be skipped during validation
// replacing the complex boolean logic with structured error classification
func IsSkippableAPIError(errorMessage string) bool {
	errorType, found := ClassifyAPIError(errorMessage)
	if !found {
		return false
	}

	// These error types are considered skippable intermittent issues
	switch errorType {
	case APIKeyError, ProjectNotFoundError, IntermittentError:
		return true
	default:
		return false
	}
}

// ConfigurationMatchStrategy defines different approaches for matching configuration names
type ConfigurationMatchStrategy int

const (
	ExactNameMatch    ConfigurationMatchStrategy = iota // Exact string match
	ContainsNameMatch                                   // String contains match (current behavior)
	BaseNameMatch                                       // Match base name without flavor/version
	PrefixNameMatch                                     // Match by prefix pattern
)

// String returns the string representation of ConfigurationMatchStrategy
func (cms ConfigurationMatchStrategy) String() string {
	switch cms {
	case ExactNameMatch:
		return "ExactNameMatch"
	case ContainsNameMatch:
		return "ContainsNameMatch"
	case BaseNameMatch:
		return "BaseNameMatch"
	case PrefixNameMatch:
		return "PrefixNameMatch"
	default:
		return "UnknownStrategy"
	}
}

// ConfigurationMatchRule defines a single matching rule with strategy and pattern
type ConfigurationMatchRule struct {
	Strategy    ConfigurationMatchStrategy
	Pattern     string // The pattern to match against
	Priority    int    // Higher priority rules are checked first
	Description string // Human-readable description of the rule
}

// IsMatch determines if the given configuration name matches this rule
func (rule *ConfigurationMatchRule) IsMatch(configName string) bool {
	switch rule.Strategy {
	case ExactNameMatch:
		return configName == rule.Pattern
	case ContainsNameMatch:
		return strings.Contains(configName, rule.Pattern)
	case BaseNameMatch:
		// Extract base name by removing flavor/version info (split on ":")
		baseName := strings.Split(rule.Pattern, ":")[0]
		return strings.Contains(configName, baseName)
	case PrefixNameMatch:
		return strings.HasPrefix(configName, rule.Pattern)
	default:
		return false
	}
}

// ConfigurationMatcher provides structured configuration name matching
// replacing fragile strings.Contains() checks with configurable matching strategies
type ConfigurationMatcher struct {
	Rules []ConfigurationMatchRule // Ordered list of matching rules (higher priority first)
}

// NewConfigurationMatcherForAddon creates a matcher configured for an addon configuration
// with appropriate fallback strategies for robust matching
func NewConfigurationMatcherForAddon(addonConfig cloudinfo.AddonConfig) *ConfigurationMatcher {
	rules := make([]ConfigurationMatchRule, 0)

	// Priority 1: Exact configuration name match (if specified)
	if addonConfig.ConfigName != "" {
		rules = append(rules, ConfigurationMatchRule{
			Strategy:    ExactNameMatch,
			Pattern:     addonConfig.ConfigName,
			Priority:    100,
			Description: fmt.Sprintf("Exact match for config name: %s", addonConfig.ConfigName),
		})
	}

	// Priority 2: Contains configuration name match (if specified)
	if addonConfig.ConfigName != "" {
		rules = append(rules, ConfigurationMatchRule{
			Strategy:    ContainsNameMatch,
			Pattern:     addonConfig.ConfigName,
			Priority:    90,
			Description: fmt.Sprintf("Contains match for config name: %s", addonConfig.ConfigName),
		})
	}

	// Priority 3: Exact offering name match
	if addonConfig.OfferingName != "" {
		rules = append(rules, ConfigurationMatchRule{
			Strategy:    ExactNameMatch,
			Pattern:     addonConfig.OfferingName,
			Priority:    80,
			Description: fmt.Sprintf("Exact match for offering name: %s", addonConfig.OfferingName),
		})
	}

	// Priority 4: Contains offering name match (current behavior)
	if addonConfig.OfferingName != "" {
		rules = append(rules, ConfigurationMatchRule{
			Strategy:    ContainsNameMatch,
			Pattern:     addonConfig.OfferingName,
			Priority:    70,
			Description: fmt.Sprintf("Contains match for offering name: %s", addonConfig.OfferingName),
		})
	}

	// Priority 5: Base offering name match (without flavor)
	if addonConfig.OfferingName != "" {
		rules = append(rules, ConfigurationMatchRule{
			Strategy:    BaseNameMatch,
			Pattern:     addonConfig.OfferingName,
			Priority:    60,
			Description: fmt.Sprintf("Base name match for offering: %s", addonConfig.OfferingName),
		})
	}

	return &ConfigurationMatcher{Rules: rules}
}

// IsMatch determines if a configuration name matches any of the configured rules
// Returns the matching rule for debugging/logging purposes
func (matcher *ConfigurationMatcher) IsMatch(configName string) (bool, *ConfigurationMatchRule) {
	// Check rules in priority order (highest priority first)
	for i := range matcher.Rules {
		rule := &matcher.Rules[i]
		if rule.IsMatch(configName) {
			return true, rule
		}
	}
	return false, nil
}

// GetBestMatch returns the highest priority matching rule for a configuration name
func (matcher *ConfigurationMatcher) GetBestMatch(configName string) *ConfigurationMatchRule {
	matched, rule := matcher.IsMatch(configName)
	if matched {
		return rule
	}
	return nil
}

// SensitiveFieldType defines different categories of sensitive data fields
type SensitiveFieldType int

const (
	APIKeyField       SensitiveFieldType = iota // API keys, tokens, credentials
	PasswordField                               // Passwords, passphrases
	SecretField                                 // Generic secrets, private keys
	CertificateField                            // Certificates, certificate data
	NonSensitiveField                           // Not sensitive data
)

// String returns the string representation of SensitiveFieldType
func (sft SensitiveFieldType) String() string {
	switch sft {
	case APIKeyField:
		return "APIKeyField"
	case PasswordField:
		return "PasswordField"
	case SecretField:
		return "SecretField"
	case CertificateField:
		return "CertificateField"
	case NonSensitiveField:
		return "NonSensitiveField"
	default:
		return "UnknownField"
	}
}

// SensitiveDataDetector provides structured detection of sensitive fields
// replacing fragile strings.Contains() checks with configurable pattern matching
type SensitiveDataDetector struct {
	SensitivePatterns map[SensitiveFieldType][]string // Patterns for each sensitivity type
}

// NewSensitiveDataDetector creates a detector with default sensitive field patterns
func NewSensitiveDataDetector() *SensitiveDataDetector {
	return &SensitiveDataDetector{
		SensitivePatterns: map[SensitiveFieldType][]string{
			APIKeyField: {
				"api_key", "apikey", "token", "access_token", "auth_token",
				"bearer_token", "oauth_token", "jwt_token", "credential",
				"credentials", "ibmcloud_api_key",
			},
			PasswordField: {
				"password", "passwd", "pwd", "passphrase", "pass",
				"user_password", "admin_password", "root_password",
			},
			SecretField: {
				"secret", "private_key", "private_key_data", "key_data",
				"encryption_key", "signing_key", "auth_secret",
			},
			CertificateField: {
				"certificate", "cert", "cert_data", "certificate_data",
				"tls_cert", "ssl_cert", "ca_cert", "ca_certificate",
			},
		},
	}
}

// ClassifyField determines the sensitivity type of a field name
func (detector *SensitiveDataDetector) ClassifyField(fieldName string) SensitiveFieldType {
	lowerFieldName := strings.ToLower(fieldName)

	// Check each sensitivity type in order of specificity
	for fieldType, patterns := range detector.SensitivePatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerFieldName, pattern) {
				return fieldType
			}
		}
	}

	return NonSensitiveField
}

// IsSensitive determines if a field contains sensitive data
func (detector *SensitiveDataDetector) IsSensitive(fieldName string) bool {
	return detector.ClassifyField(fieldName) != NonSensitiveField
}

// ShouldLogValue determines if a field value should be logged based on sensitivity
func (detector *SensitiveDataDetector) ShouldLogValue(fieldName string) bool {
	return !detector.IsSensitive(fieldName)
}

// GetMaskedValue returns a masked representation of sensitive values for safe logging
func (detector *SensitiveDataDetector) GetMaskedValue(fieldName string, value interface{}) string {
	if !detector.IsSensitive(fieldName) {
		return fmt.Sprintf("%v", value)
	}

	fieldType := detector.ClassifyField(fieldName)
	switch fieldType {
	case APIKeyField:
		return "[API_KEY_REDACTED]"
	case PasswordField:
		return "[PASSWORD_REDACTED]"
	case SecretField:
		return "[SECRET_REDACTED]"
	case CertificateField:
		return "[CERTIFICATE_REDACTED]"
	default:
		return "[SENSITIVE_DATA_REDACTED]"
	}
}

// Default detector instance for package-wide use
var defaultSensitiveDataDetector = NewSensitiveDataDetector()

// IsSensitiveField is a package-level convenience function for sensitivity checking
func IsSensitiveField(fieldName string) bool {
	return defaultSensitiveDataDetector.IsSensitive(fieldName)
}

// GetSafeMaskedValue is a package-level convenience function for safe value masking
func GetSafeMaskedValue(fieldName string, value interface{}) string {
	return defaultSensitiveDataDetector.GetMaskedValue(fieldName, value)
}

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

	// ProjectAutoDeployMode Valid values are "manual_approval" and "auto_approval".
	ProjectAutoDeployMode string
	ProjectEnvironments   []project.EnvironmentPrototype

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

	// ProjectRetryConfig Configuration for project creation/deletion retry behavior (optional)
	// When nil, uses common.ProjectOperationRetryConfig() defaults (5 retries, 3s initial delay, 45s max, exponential backoff)
	ProjectRetryConfig *common.RetryConfig
	// CatalogRetryConfig Configuration for catalog operation retry behavior (optional)
	// When nil, uses common.CatalogOperationRetryConfig() defaults (5 retries, 3s initial delay, 30s max, linear backoff)
	CatalogRetryConfig *common.RetryConfig
	// DeployRetryConfig Configuration for deployment operation retry behavior (optional)
	// When nil, uses common.DefaultRetryConfig() defaults (3 retries, 2s initial delay, 30s max, exponential backoff)
	DeployRetryConfig *common.RetryConfig

	// StaggerDelay Configuration for delay between starting batches of parallel tests (optional)
	// When nil, uses default 10 seconds. Set to 0 to disable staggering.
	// Recommended values: 5-15 seconds for most scenarios, 20-30 seconds for high API sensitivity.
	StaggerDelay *time.Duration
	// StaggerBatchSize Configuration for number of tests per batch for staggered execution (optional)
	// When nil, uses default 8 tests per batch. Set to 0 to use linear staggering.
	// Recommended values: 8-12 for default, 4-6 for high API sensitivity, 15-25 for low sensitivity.
	StaggerBatchSize *int
	// WithinBatchDelay Configuration for delay between tests within the same batch (optional)
	// When nil, uses default 2 seconds. Only used when StaggerBatchSize > 0.
	// Recommended values: 1-3 seconds for most scenarios, 5+ for high sensitivity.
	WithinBatchDelay *time.Duration

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

	// StrictMode controls validation behavior for circular dependencies and required dependency force-enabling.
	//
	// When true (default):
	//   - Circular dependencies cause test failure
	//   - Force-enabled required dependencies generate warnings but test continues
	//
	// When false (permissive mode):
	//   - Circular dependencies are logged as warnings, test continues
	//   - Required dependencies are force-enabled silently with informational messages
	//   - Warnings are captured and displayed in final permutation test report
	//   - Final report shows "STRICT MODE DISABLED" section with warnings that would have failed in strict mode
	//
	// Use StrictMode=false for dependency permutation testing where you need to test
	// all combinations while understanding what would fail in production (strict mode).
	// The final report will clearly show which scenarios would be problematic in strict mode.
	StrictMode *bool

	// OverrideInputMappings If set to false (default), preserves existing reference values (starting with "ref:") when merging inputs.
	// When true, uses current behavior and overrides all input values regardless of whether they are references.
	// This allows controlled preservation of input mappings that reference other configuration outputs.
	OverrideInputMappings *bool

	// internal use
	configInputReferences map[string]map[string]string // ConfigID -> FieldName -> ReferenceValue cache
	currentProject        *project.Project
	currentProjectConfig  *cloudinfo.ProjectsConfig
	deployedConfigs       *cloudinfo.DeployedAddonsDetails // Store deployed configs for validation

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
	lastTeardownErrors   []string

	// PostCreateDelay is the delay to wait after creating resources before attempting to read them.
	// This helps with eventual consistency issues in IBM Cloud APIs.
	// Default: 1 second. Set to a pointer to 0 duration to disable delays explicitly.
	PostCreateDelay *time.Duration

	// GetDirectDependencyNames allows test injection of dependency names for permutation testing
	// When set, this function will be called instead of reading from ibm_catalog.json
	// Used primarily for mocking dependencies in comprehensive regression tests
	GetDirectDependencyNames func() ([]string, error)

	// SkipPermutations allows temporarily skipping specific permutation test cases.
	// Each inner slice defines the set of ENABLED dependencies for a permutation to skip.
	// - Use full offering and flavor names (flavor optional = wildcard)
	// - Order does not matter; comparison is set-based
	// - Dependencies not listed are treated as DISABLED in that permutation
	// Example: skip enabling cloud-logs[rgo] + kms[instance] only (others disabled):
	//   []cloudinfo.AddonConfig{
	//       {OfferingName: "deploy-arch-ibm-cloud-logs", OfferingFlavor: "resource-group-only"},
	//       {OfferingName: "deploy-arch-ibm-kms", OfferingFlavor: "instance"},
	//   }
	SkipPermutations [][]cloudinfo.AddonConfig

	// CacheEnabled enables API response caching for catalog operations to reduce API calls by 70-80%
	// When enabled, static catalog metadata (offerings, versions, dependencies) will be cached
	// Dynamic state (configs, deployments, validation) is never cached to ensure test correctness
	// Default: true (cache enabled by default for performance benefits)
	CacheEnabled *bool

	// CacheTTL sets the time-to-live for cached API responses
	// Default: 10 minutes if not specified when cache is enabled
	// Recommended: 5-15 minutes for test scenarios, 10 minutes for CI/CD pipelines
	CacheTTL time.Duration
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
	if newOptions.ProjectAutoDeployMode == "" {
		newOptions.ProjectAutoDeployMode = project.ProjectDefinition_AutoDeployMode_AutoApproval
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

	// Set default StrictMode to true (fail on required dependency validation issues)
	if newOptions.StrictMode == nil {
		newOptions.StrictMode = core.BoolPtr(true)
	}

	// Set default OverrideInputMappings to false (preserve reference values)
	if newOptions.OverrideInputMappings == nil {
		newOptions.OverrideInputMappings = core.BoolPtr(false)
	}

	// Set default EnhancedTreeValidationOutput to true (show dependency trees for better debugging)
	if !newOptions.EnhancedTreeValidationOutput {
		newOptions.EnhancedTreeValidationOutput = true
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

	// Set default post-creation delay if not already set
	if newOptions.PostCreateDelay == nil {
		delay := 1 * time.Second
		newOptions.PostCreateDelay = &delay
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

// copyAddonConfig creates a deep copy of AddonConfig to avoid reference sharing
// This is critical for matrix tests where each test case needs independent dependency configurations
func copyAddonConfig(original cloudinfo.AddonConfig) cloudinfo.AddonConfig {
	copied := original // Start with shallow copy of all fields

	// Deep copy the Dependencies slice to avoid sharing references between test cases
	if original.Dependencies != nil {
		copied.Dependencies = make([]cloudinfo.AddonConfig, len(original.Dependencies))
		for i, dep := range original.Dependencies {
			copied.Dependencies[i] = copyAddonConfig(dep) // Recursive deep copy
		}
	}

	// Deep copy the Inputs map to avoid sharing references
	if original.Inputs != nil {
		copied.Inputs = make(map[string]interface{})
		for k, v := range original.Inputs {
			copied.Inputs[k] = v
		}
	}

	// Deep copy pointer fields to avoid sharing
	if original.Enabled != nil {
		enabled := *original.Enabled
		copied.Enabled = &enabled
	}

	if original.OnByDefault != nil {
		onByDefault := *original.OnByDefault
		copied.OnByDefault = &onByDefault
	}

	if original.IsRequired != nil {
		isRequired := *original.IsRequired
		copied.IsRequired = &isRequired
	}

	// Deep copy RequiredBy slice
	if original.RequiredBy != nil {
		copied.RequiredBy = make([]string, len(original.RequiredBy))
		copy(copied.RequiredBy, original.RequiredBy)
	}

	return copied
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
		ProjectAutoDeployMode:        options.ProjectAutoDeployMode,
		ProjectEnvironments:          options.ProjectEnvironments,
		CloudInfoService:             options.CloudInfoService,
		CatalogUseExisting:           options.CatalogUseExisting,
		CatalogName:                  options.CatalogName,
		SharedCatalog:                copyBoolPointer(options.SharedCatalog),
		AddonConfig:                  copyAddonConfig(options.AddonConfig), // Deep copy to avoid reference sharing
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
		PostCreateDelay:              options.PostCreateDelay,
		ProjectRetryConfig:           options.ProjectRetryConfig,
		CatalogRetryConfig:           options.CatalogRetryConfig,
		DeployRetryConfig:            options.DeployRetryConfig,
		StaggerDelay:                 options.StaggerDelay,
		StaggerBatchSize:             options.StaggerBatchSize,
		WithinBatchDelay:             options.WithinBatchDelay,
		PreDeployHook:                options.PreDeployHook,
		PostDeployHook:               options.PostDeployHook,
		PreUndeployHook:              options.PreUndeployHook,
		PostUndeployHook:             options.PostUndeployHook,
		Logger:                       nil, // Force creation of unique logger per test to avoid cross-contamination in parallel tests
		QuietMode:                    options.QuietMode,
		StrictMode:                   copyBoolPointer(options.StrictMode),
		OverrideInputMappings:        copyBoolPointer(options.OverrideInputMappings),

		// Result collection fields need to be shared across all test instances
		CollectResults:        options.CollectResults,
		PermutationTestReport: options.PermutationTestReport,

		// These fields are not copied as they are managed per test instance
		configInputReferences: nil,
		catalog:               nil,
		offering:              nil,
		currentProject:        nil,
		currentProjectConfig:  nil,
		deployedConfigs:       nil,
		currentBranch:         nil,
		currentBranchUrl:      nil,
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
	// Create base result preserving the hierarchical addon configuration structure
	// Main addon contains its dependencies in the Dependencies field (tree structure)

	result := PermutationTestResult{
		Name:        testName,
		Prefix:      testPrefix,
		AddonConfig: []cloudinfo.AddonConfig{addonConfig}, // Preserve tree structure with nested dependencies
		Passed:      testError == nil && !options.Testing.Failed(),
		StrictMode:  options.StrictMode,
	}

	// Collect validation errors if available
	if options.lastValidationResult != nil {
		result.ValidationResult = options.lastValidationResult

		// Extract strict mode warnings when running in permissive mode
		if options.StrictMode != nil && !*options.StrictMode && options.lastValidationResult.Warnings != nil {
			result.StrictModeWarnings = append(result.StrictModeWarnings, options.lastValidationResult.Warnings...)
		}
	}

	// Collect other error categories (simplified)
	if options.lastTransientErrors != nil {
		result.TransientErrors = append(result.TransientErrors, options.lastTransientErrors...)
	}

	if options.lastRuntimeErrors != nil {
		result.RuntimeErrors = append(result.RuntimeErrors, options.lastRuntimeErrors...)
	}

	// Add teardown errors to RuntimeErrors for reporting
	if options.lastTeardownErrors != nil {
		result.RuntimeErrors = append(result.RuntimeErrors, options.lastTeardownErrors...)
	}

	// If test failed, parse and categorize the main error
	if testError != nil {
		options.categorizeError(testError, &result)
	}

	// Reset error collection fields for next test
	options.lastValidationResult = nil
	options.lastTransientErrors = nil
	options.lastRuntimeErrors = nil
	options.lastTeardownErrors = nil

	return result
}

// categorizeError parses the main test error and categorizes it into one of three simplified categories
func (options *TestAddonOptions) categorizeError(testError error, result *PermutationTestResult) {
	// Check if we already have detailed error info
	hasDetailedErrors := (result.ValidationResult != nil && !result.ValidationResult.IsValid) ||
		len(result.TransientErrors) > 0 || len(result.RuntimeErrors) > 0

	// Only categorize if we don't have detailed errors AND haven't already categorized this result
	// This prevents double processing of errors
	if !hasDetailedErrors && !result.ErrorAlreadyCategorized {
		result.ErrorAlreadyCategorized = true
		options.categorizeMainError(testError, result)
	}
}

// categorizeMainError contains the core error categorization logic using structured patterns
// replacing fragile strings.Contains() checks with regex-based classification
func (options *TestAddonOptions) categorizeMainError(testError error, result *PermutationTestResult) {
	errorStr := testError.Error()

	// Use structured pattern matching instead of hardcoded switch statement
	if pattern, found := classifyError(errorStr); found {
		switch pattern.Type {
		case ValidationError:
			options.addValidationError(result, errorStr, pattern.Subtype)
		case TransientError:
			result.TransientErrors = append(result.TransientErrors, errorStr)
		case RuntimeError:
			result.RuntimeErrors = append(result.RuntimeErrors, errorStr)
		}
	} else {
		// Default to transient error for unknown issues (likely infrastructure)
		result.TransientErrors = append(result.TransientErrors, errorStr)
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
	target.Warnings = append(target.Warnings, source.Warnings...)
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
