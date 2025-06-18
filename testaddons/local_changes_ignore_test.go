package testaddons

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultIgnorePatternsIncludeSubmodule tests that the default ignore patterns
// include the pattern to ignore git submodule pointer changes for common-dev-assets
func TestDefaultIgnorePatternsIncludeSubmodule(t *testing.T) {
	// Create a test options instance with defaults
	originalOptions := &TestAddonOptions{
		Testing: t,
		Prefix:  "test",
	}

	options := TestAddonsOptionsDefault(originalOptions)

	// Check that the default ignore patterns include both submodule patterns
	foundExactMatch := false
	foundDirectoryMatch := false

	for _, pattern := range options.LocalChangesIgnorePattern {
		if pattern == "^common-dev-assets$" {
			foundExactMatch = true
		}
		if pattern == "^common-dev-assets/.*" {
			foundDirectoryMatch = true
		}
	}

	assert.True(t, foundExactMatch, "Default ignore patterns should include '^common-dev-assets$' for submodule pointer changes")
	assert.True(t, foundDirectoryMatch, "Default ignore patterns should include '^common-dev-assets/.*' for files within the directory")
}

// TestIgnorePatternSubmoduleMatch tests that the submodule ignore pattern
// correctly matches the exact submodule name
func TestIgnorePatternSubmoduleMatch(t *testing.T) {
	pattern := "^common-dev-assets$"

	// Test cases
	testCases := []struct {
		filename string
		expected bool
	}{
		{"common-dev-assets", true},                  // exact match - should be ignored
		{"common-dev-assets/file.txt", false},        // directory/file - should NOT match this pattern
		{"common-dev-assets/subdir/file.txt", false}, // nested file - should NOT match this pattern
		{"other-common-dev-assets", false},           // different name - should NOT match
		{"common-dev-assets-old", false},             // different name - should NOT match
		{"someother/common-dev-assets", false},       // in different directory - should NOT match
	}

	for _, tc := range testCases {
		matched, err := regexp.MatchString(pattern, tc.filename)
		assert.NoError(t, err, "Regex should not error for: %s", tc.filename)
		assert.Equal(t, tc.expected, matched, "Pattern '%s' match result for '%s' should be %v", pattern, tc.filename, tc.expected)
	}
}

// TestIgnorePatternDirectoryMatch tests that the directory ignore pattern
// correctly matches files within the common-dev-assets directory
func TestIgnorePatternDirectoryMatch(t *testing.T) {
	pattern := "^common-dev-assets/.*"

	// Test cases
	testCases := []struct {
		filename string
		expected bool
	}{
		{"common-dev-assets", false},                // exact match - should NOT match this pattern
		{"common-dev-assets/file.txt", true},        // directory/file - should be ignored
		{"common-dev-assets/subdir/file.txt", true}, // nested file - should be ignored
		{"other-common-dev-assets", false},          // different name - should NOT match
		{"common-dev-assets-old", false},            // different name - should NOT match
		{"someother/common-dev-assets", false},      // in different directory - should NOT match
		{"common-dev-assets/", true},                // directory with trailing slash - should match
	}

	for _, tc := range testCases {
		matched, err := regexp.MatchString(pattern, tc.filename)
		assert.NoError(t, err, "Regex should not error for: %s", tc.filename)
		assert.Equal(t, tc.expected, matched, "Pattern '%s' match result for '%s' should be %v", pattern, tc.filename, tc.expected)
	}
}

// TestBothPatternsTogetherCoverAllCases tests that both patterns together
// cover all common-dev-assets related cases that should be ignored
func TestBothPatternsTogetherCoverAllCases(t *testing.T) {
	patterns := []string{"^common-dev-assets$", "^common-dev-assets/.*"}

	// Test cases that should all be ignored
	shouldBeIgnored := []string{
		"common-dev-assets",                 // submodule pointer change
		"common-dev-assets/file.txt",        // file in directory
		"common-dev-assets/subdir/file.txt", // nested file
		"common-dev-assets/",                // directory with trailing slash
	}

	// Test cases that should NOT be ignored
	shouldNotBeIgnored := []string{
		"other-common-dev-assets",     // different name
		"common-dev-assets-old",       // different name
		"someother/common-dev-assets", // in different directory
		"test.txt",                    // completely different file
	}

	for _, filename := range shouldBeIgnored {
		matched := false
		for _, pattern := range patterns {
			if m, err := regexp.MatchString(pattern, filename); err == nil && m {
				matched = true
				break
			}
		}
		assert.True(t, matched, "File '%s' should be ignored by at least one pattern", filename)
	}

	for _, filename := range shouldNotBeIgnored {
		matched := false
		for _, pattern := range patterns {
			if m, err := regexp.MatchString(pattern, filename); err == nil && m {
				matched = true
				break
			}
		}
		assert.False(t, matched, "File '%s' should NOT be ignored by any pattern", filename)
	}
}
