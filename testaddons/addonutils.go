package testaddons

import (
	"fmt"
	"strings"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// Common constants for error patterns and tree formatting
const (
	// Error message patterns
	MissingInputsPattern        = "missing required inputs:"
	MissingInputsDetailPattern  = " (missing: "
	DependencyValidationPattern = "dependency validation failed:"
	UnexpectedConfigsPattern    = "unexpected configs"
	ShouldNotBeDeployedPattern  = "should not be deployed"

	// Tree formatting symbols
	TreeBranch     = "├── "
	TreeLastBranch = "└── "
	TreeVertical   = "│   "
	TreeSpace      = "    "

	// Status indicators
	StatusPassed  = "✅"
	StatusFailed  = "❌"
	StatusWarning = "⚠️"
	StatusInfo    = "ℹ️"
)

// Key Generation Utils
// generateAddonKey creates a consistent key for addon identification across the codebase
// This replaces 23+ instances of fmt.Sprintf("%s:%s:%s", name, version, flavor)
func generateAddonKey(name, version, flavor string) string {
	return fmt.Sprintf("%s:%s:%s", name, version, flavor)
}

// generateAddonKeyFromDetail creates a key from OfferingReferenceDetail
func generateAddonKeyFromDetail(detail cloudinfo.OfferingReferenceDetail) string {
	return generateAddonKey(detail.Name, detail.Version, detail.Flavor.Name)
}

// generateAddonKeyFromDependencyError creates a key from DependencyError
func generateAddonKeyFromDependencyError(depErr cloudinfo.DependencyError) string {
	return generateAddonKey(depErr.Addon.Name, depErr.Addon.Version, depErr.Addon.Flavor.Name)
}

// Error Parsing Utils
// ErrorComponents represents the components extracted from an error message
type ErrorComponents struct {
	ConfigName string
	Version    string
	Flavor     string
	InputName  string
}

// extractAllErrorComponents extracts all available components from an error message
// Uses existing functions from test_options.go for consistency
func extractAllErrorComponents(errorStr string) ErrorComponents {
	return ErrorComponents{
		ConfigName: extractConfigNameFromError(errorStr),
		Version:    extractVersionFromError(errorStr),
		Flavor:     extractFlavorFromError(errorStr),
	}
}

// Tree Traversal Utils
// TreeTraversalOptions configures tree printing behavior
type TreeTraversalOptions struct {
	ShowStatus      bool
	ShowValidation  bool
	ShowPath        bool
	MaxDepth        int
	IncludeWarnings bool
	CompactMode     bool
}

// DefaultTreeOptions returns sensible defaults for tree traversal
func DefaultTreeOptions() TreeTraversalOptions {
	return TreeTraversalOptions{
		ShowStatus:      true,
		ShowValidation:  false,
		ShowPath:        false,
		MaxDepth:        10,
		IncludeWarnings: true,
		CompactMode:     false,
	}
}

// formatTreeSymbol returns the appropriate tree symbol based on position
func formatTreeSymbol(isLast bool) string {
	if isLast {
		return TreeLastBranch
	}
	return TreeBranch
}

// formatTreeIndent returns the appropriate indentation based on position
func formatTreeIndent(isLast bool) string {
	if isLast {
		return TreeSpace
	}
	return TreeVertical
}

// checkCircularReference checks if we're in a circular dependency and handles visited tracking
func checkCircularReference(key string, visited map[string]bool) (bool, func()) {
	if visited[key] {
		return true, func() {} // No cleanup needed for circular reference
	}

	visited[key] = true
	cleanup := func() {
		delete(visited, key)
	}

	return false, cleanup
}

// Common Validation Utils
// isValidationError checks if an error string indicates a validation issue
// Uses the new structured ErrorPattern system from test_options.go for consistency
func isValidationError(errorStr string) bool {
	if pattern, found := classifyError(errorStr); found {
		return pattern.Type == ValidationError
	}
	return false
}

// isTransientError checks if an error string indicates a transient/infrastructure issue
// Uses the new structured ErrorPattern system from test_options.go for consistency
func isTransientError(errorStr string) bool {
	if pattern, found := classifyError(errorStr); found {
		return pattern.Type == TransientError
	}
	return false
}

// isRuntimeError checks if an error string indicates a runtime/code issue
// Uses the new structured ErrorPattern system from test_options.go for consistency
func isRuntimeError(errorStr string) bool {
	if pattern, found := classifyError(errorStr); found {
		return pattern.Type == RuntimeError
	}
	return false
}

// Pattern Extraction Utils
// isAlphanumeric checks if a string contains only alphanumeric characters
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// normalizeConfigName removes common suffixes from config names for pattern matching
func normalizeConfigName(configName string) string {
	// Remove trailing random IDs (pattern: config-name-abc123)
	parts := strings.Split(configName, "-")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// If last part looks like a random ID (6 alphanumeric chars), remove it
		if len(lastPart) == 6 && isAlphanumeric(lastPart) {
			return strings.Join(parts[:len(parts)-1], "-")
		}
	}

	return configName
}

// Dependency Helper Utils
// FindDependencyByName finds a dependency by name in a slice of AddonConfig dependencies
func FindDependencyByName(dependencies []cloudinfo.AddonConfig, name string) *cloudinfo.AddonConfig {
	for i, dep := range dependencies {
		if dep.OfferingName == name {
			return &dependencies[i]
		}
	}
	return nil
}

// ExtractDependencyNames extracts the offering names from a slice of AddonConfig dependencies
func ExtractDependencyNames(dependencies []cloudinfo.AddonConfig) []string {
	names := make([]string, len(dependencies))
	for i, dep := range dependencies {
		names[i] = dep.OfferingName
	}
	return names
}
