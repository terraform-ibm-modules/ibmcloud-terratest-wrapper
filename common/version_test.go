package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input         string
		expectedMajor int
		expectedMinor int
		expectedPatch int
		expectedValid bool
	}{
		{"v1.2.3", 1, 2, 3, true},
		{"v0.0.1", 0, 0, 1, true},
		{"1.2.3", 0, 0, 0, false},    // missing 'v'
		{"v1.2", 0, 0, 0, false},     // incomplete version
		{"v1.2.3.4", 0, 0, 0, false}, // too many parts
		{"vx.y.z", 0, 0, 0, false},   // non-numeric
	}

	for _, tt := range tests {
		major, minor, patch, valid := parseSemver(tt.input)
		assert.Equal(t, tt.expectedMajor, major, "major mismatch for input %s", tt.input)
		assert.Equal(t, tt.expectedMinor, minor, "minor mismatch for input %s", tt.input)
		assert.Equal(t, tt.expectedPatch, patch, "patch mismatch for input %s", tt.input)
		assert.Equal(t, tt.expectedValid, valid, "validity mismatch for input %s", tt.input)
	}
}

func TestMatchVersion(t *testing.T) {
	versions := []string{
		"v1.2.3", "v1.3.0", "v1.2.4", "v2.0.0", "v3.1.5", "v3.0.0", "v3.0.1",
	}

	tests := []struct {
		target      string
		expected    string
		description string
	}{
		{"v1.2.3", "v1.2.3", "exact match"},
		{"v1.2.5", "", "no exact match"},
		{"^v3.0.0", "v3.1.5", "caret match should allow higher minor/patch in same major"},
		{"^v1.2.0", "v1.3.0", "caret match within major"},
		{"~v1.2.0", "v1.2.4", "tilde match within same minor"},
		{"~v2.0.0", "v2.0.0", "tilde match exact version"},
		{"~v4.0.0", "", "no tilde match found"},
		{"invalid", "", "invalid version"},
	}

	for _, tt := range tests {
		result := MatchVersion(versions, tt.target)
		assert.Equal(t, tt.expected, result, tt.description)
	}
}
