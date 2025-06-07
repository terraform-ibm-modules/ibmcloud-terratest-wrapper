package common

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// parseSemver function is used by MatchVersion function to find the most suitable version
// in case it is not pinned in the dependency
func parseSemver(version string) (major int, minor int, fix int, valid bool) {
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(version)
	if matches == nil {
		return 0, 0, 0, false
	}
	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	fix, _ = strconv.Atoi(matches[3])
	return major, minor, fix, true
}

// MatchVersion function takes all the versions available for a dependency
// and returns the suitable version matching target.
// Target can be an exact version (e.g., "v1.2.3"),
// unpinned version with `^` (e.g., "^v3.0.1" for major match, latest minor/patch),
// unpinned version with `~` (e.g., "~v4.1.4" for major and minor match, latest patch),
// or a range (e.g., ">=v1.2.3,<=v7.1.3" or "<=v4.2.1,>=v1.1.1").
// The "v" keyword is mandatory for all version numbers.
// Returns an empty string if no suitable version is found or the target format is invalid.
func MatchVersion(versions []string, target string) string {
	var matchTargetType string                    // "exact", "^", "~", "range"
	var targetMajor, targetMinor, targetPatch int // for exact, ^, ~ matches
	var minMajor, minMinor, minPatch int          // for range matches
	var maxMajor, maxMinor, maxPatch int          // for range matches

	// Regular expressions for target validation and parsing
	reExactPrefix := regexp.MustCompile(`^([~^])?(v\d+\.\d+\.\d+)$`)
	reRangePart := regexp.MustCompile(`^(>=|<=)(v\d+\.\d+\.\d+)$`)

	if strings.Contains(target, ",") { // Potential range match
		parts := strings.Split(target, ",")
		if len(parts) != 2 {
			return "" // Invalid range format: must have exactly two parts separated by comma
		}

		match1 := reRangePart.FindStringSubmatch(parts[0])
		match2 := reRangePart.FindStringSubmatch(parts[1])

		if match1 == nil || match2 == nil {
			return "" // Invalid range part format
		}

		op1, verStr1 := match1[1], match1[2]
		op2, verStr2 := match2[1], match2[2]

		v1Major, v1Minor, v1Patch, ok1 := parseSemver(verStr1)
		v2Major, v2Minor, v2Patch, ok2 := parseSemver(verStr2)

		if !ok1 || !ok2 {
			return "" // Versions in range parts are invalid semver
		}

		// Validate that one operator is '>=' and the other is '<='
		if (op1 == ">=" && op2 == ">=") || (op1 == "<=" && op2 == "<=") {
			return "" // Invalid range: both parts specify min or both specify max
		}

		matchTargetType = "range"

		// Assign min and max versions based on the operators
		if op1 == ">=" {
			minMajor, minMinor, minPatch = v1Major, v1Minor, v1Patch
			maxMajor, maxMinor, maxPatch = v2Major, v2Minor, v2Patch
		} else { // op1 == "<="
			minMajor, minMinor, minPatch = v2Major, v2Minor, v2Patch
			maxMajor, maxMinor, maxPatch = v1Major, v1Minor, v1Patch
		}

		// Additional validation: ensure min version is logically less than or equal to max version
		// If min > max, the range is empty.
		if minMajor > maxMajor ||
			(minMajor == maxMajor && minMinor > maxMinor) ||
			(minMajor == maxMajor && minMinor == maxMinor && minPatch > maxPatch) {
			return "" // Empty or invalid range (e.g., >=v5.0.0,<=v1.0.0)
		}

	} else { // Exact or prefixed match (`^` or `~`)
		matches := reExactPrefix.FindStringSubmatch(target)
		if matches == nil {
			return "" // Invalid target format: does not match exact, ^, ~, or range
		}

		prefixOp := matches[1]     // Will be "" for exact, or "~", or "^"
		targetVerStr := matches[2] // e.g., "v1.2.3"

		var ok bool
		targetMajor, targetMinor, targetPatch, ok = parseSemver(targetVerStr)
		if !ok {
			return "" // This should be caught by regex, but for safety.
		}

		if prefixOp == "^" {
			matchTargetType = "^"
		} else if prefixOp == "~" {
			matchTargetType = "~"
		} else {
			matchTargetType = "exact"
		}
	}

	candidates := [][]int{}
	versionMap := map[string][]int{} // Maps original version string to its triplet

	for _, v := range versions {
		major, minor, patch, valid := parseSemver(v)
		if !valid {
			continue // Skip invalid available versions
		}
		versionTriplet := []int{major, minor, patch}
		versionMap[v] = versionTriplet

		// Handle version matching based on matchTargetType
		switch matchTargetType {
		case "exact":
			if major == targetMajor && minor == targetMinor && patch == targetPatch {
				return v // For exact matches, return immediately upon finding
			}
		case "^": // Major version must match, find latest minor/patch
			if major == targetMajor && (minor > targetMinor || (minor == targetMinor && patch >= targetPatch)) {
				candidates = append(candidates, versionTriplet)
			}
		case "~": // Major and minor versions must match, find latest patch
			if major == targetMajor && minor == targetMinor && patch >= targetPatch {
				candidates = append(candidates, versionTriplet)
			}
		case "range":
			// Inline comparison logic for current version 'v' against min and max bounds
			// Check if v >= minVersion
			isGreaterThanOrEqualToMin := false
			if major > minMajor {
				isGreaterThanOrEqualToMin = true
			} else if major == minMajor {
				if minor > minMinor {
					isGreaterThanOrEqualToMin = true
				} else if minor == minMinor {
					if patch >= minPatch {
						isGreaterThanOrEqualToMin = true
					}
				}
			}

			// Check if v <= maxVersion
			isLessThanOrEqualToMax := false
			if major < maxMajor {
				isLessThanOrEqualToMax = true
			} else if major == maxMajor {
				if minor < maxMinor {
					isLessThanOrEqualToMax = true
				} else if minor == maxMinor {
					if patch <= maxPatch {
						isLessThanOrEqualToMax = true
					}
				}
			}

			if isGreaterThanOrEqualToMin && isLessThanOrEqualToMax {
				candidates = append(candidates, versionTriplet)
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by major, minor, patch in descending order to find the largest
	sort.SliceStable(candidates, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if candidates[i][k] != candidates[j][k] {
				return candidates[i][k] > candidates[j][k]
			}
		}
		return false
	})

	// Convert the top candidate (largest version) back to its original string format
	top := candidates[0]
	for ver, parts := range versionMap {
		if parts[0] == top[0] && parts[1] == top[1] && parts[2] == top[2] {
			return ver // Return the original version string
		}
	}

	return ""
}
