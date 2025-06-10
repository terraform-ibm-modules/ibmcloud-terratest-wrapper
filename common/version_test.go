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
		"v0.0.1", "0.1.3", "v1.1.1", "1.2.3", "v1.2.4", "v1.3.0", "v2.0.0",
		"2.1.1", "v2.2.0", "v3.0.0", "v3.0.1", "v3.1.4", "v3.1.5",
		"v4.0.0", "v4.0.4", "v4.3.1", "5.0.0", "v7.2.1", "v9.0.0",
		"v9.1.2", "v9.1.4", "v10.3.1",
	}

	tests := []struct {
		target      string
		expected    string
		description string
	}{
		{"v1.2.3", "1.2.3", "exact match"},
		{"v1.2.5", "", "no exact match"},
		{"^v3.0.0", "3.1.5", "caret match should allow higher minor/patch in same major"},
		{"^v1.2.0", "1.3.0", "caret match within major"},
		{"^2.1.2", "2.2.0", "caret match within major 2"},
		{"^v4.0.0", "4.3.1", "caret match with patch and minor bump"},
		{"~v1.2.0", "1.2.4", "tilde match within same minor"},
		{"~v2.0.0", "2.0.0", "tilde match exact version"},
		{"~v3.1.0", "3.1.5", "tilde match within 3.1.x"},
		{"~4.0.0", "4.0.4", "tilde match 4.0.x"},
		{"~v5.0.0", "5.0.0", "tilde match exact version, no patch bump"},
		{">=v1.1.1,<=v3.1.4", "3.1.4", "range match up to v3.1.4"},
		{"<=v1.1.1,>=v0.0.0", "1.1.1", "range match from v0.0.0 to v1.1.1"},
		{">=v2.1.0,<=v2.2.0", "2.2.0", "range within v2.1.0 to v2.2.0"},
		{">=v9.0.0,<=v9.1.4", "9.1.4", "range match within major 9"},
		{">=v10.0.0,<=v10.3.2", "10.3.1", "range match within v10.x"},
		{"invalid", "", "invalid version"},
		{">=v3.2.0,<=v8.5.0", "7.2.1", "no match in range"},
		{">=3.2.0", "10.3.1", "upper range found"},
		{"<=0.2.0", "0.1.3", "lower range  found"},
	}

	for _, tt := range tests {
		result := MatchVersion(versions, tt.target)
		assert.Equal(t, tt.expected, result, tt.description)
	}
}
