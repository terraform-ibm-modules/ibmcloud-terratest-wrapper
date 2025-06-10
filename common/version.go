package common

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// parseSemver function is used by MatchVersion function to find the most suitable version
// in case it is not pinned in the dependency. It expects a version string without "v" prefix.
func parseSemver(version string) (major int, minor int, fix int, valid bool) {
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
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
// and returns the suitable version matching target. The "v" keyword is optional
// and will be stripped from both input versions and the target.
// All returned versions will also be without the "v" prefix.
//
// Target can be:
// - An exact version (e.g., "1.2.3")
// - Unpinned version with `^` (e.g., "^3.0.1" for major match, latest minor/patch)
// - Unpinned version with `~` (e.g., "~4.1.4" for major and minor match, latest patch)
// - A two-sided range (e.g., ">=1.2.3,<=7.1.3" or "<=4.2.1,>=1.1.1")
// - A single-sided range (e.g., ">=1.2.3" or "<=4.5.6")
//
// Returns an empty string if no suitable version is found or the target format is invalid.
func MatchVersion(versions []string, target string) string {
	var matchTargetType string                    // "exact", "^", "~", "range", "min_range", "max_range"
	var targetMajor, targetMinor, targetPatch int // for exact, ^, ~ matches
	var minMajor, minMinor, minPatch int          // for range matches
	var maxMajor, maxMinor, maxPatch int          // for range matches

	// Helper to clean version strings (strip 'v' and trim spaces)
	cleanVersion := func(ver string) string {
		ver = strings.ReplaceAll(ver, " ", "")
		return strings.ReplaceAll(ver, "v", "")
	}

	// Clean the input versions slice in place
	for i := range versions {
		versions[i] = cleanVersion(versions[i])
	}

	// Clean the target string in place
	target = cleanVersion(target)

	// Regular expressions for target validation and parsing
	reExactPrefix := regexp.MustCompile(`^([~^])?(\d+\.\d+\.\d+)$`)
	reRangePart := regexp.MustCompile(`^(>=|<=)(\d+\.\d+\.\d+)$`)

	if strings.Contains(target, ",") { // Potential two-sided range match
		parts := strings.Split(target, ",")
		if len(parts) != 2 {
			return "" // Invalid range format: must have exactly one comma for two parts
		}

		match1 := reRangePart.FindStringSubmatch(parts[0])
		match2 := reRangePart.FindStringSubmatch(parts[1])

		if match1 == nil || match2 == nil {
			return "" // Invalid range part format
		}

		op1, verStr1 := match1[1], match1[2]
		op2, verStr2 := match2[1], match2[2]

		v1Major, v1Minor, v11Patch, ok1 := parseSemver(verStr1)
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
			minMajor, minMinor, minPatch = v1Major, v1Minor, v11Patch
			maxMajor, maxMinor, maxPatch = v2Major, v2Minor, v2Patch
		} else { // op1 == "<="
			minMajor, minMinor, minPatch = v2Major, v2Minor, v2Patch
			maxMajor, maxMinor, maxPatch = v1Major, v1Minor, v11Patch
		}

		// Additional validation: ensure min version is logically less than or equal to max version
		// If min > max, the range is empty.
		if minMajor > maxMajor ||
			(minMajor == maxMajor && minMinor > maxMinor) ||
			(minMajor == maxMajor && minMinor == maxMinor && minPatch > maxPatch) {
			return "" // Empty or invalid range (e.g., >=5.0.0,<=1.0.0)
		}

	} else { // Exact, prefixed match (`^` or `~`), or single-sided range
		if strings.HasPrefix(target, ">=") {
			match := reRangePart.FindStringSubmatch(target)
			if match == nil {
				return "" // Invalid format for single-sided range
			}
			matchTargetType = "min_range"
			minMajor, minMinor, minPatch, _ = parseSemver(match[2])
		} else if strings.HasPrefix(target, "<=") {
			match := reRangePart.FindStringSubmatch(target)
			if match == nil {
				return "" // Invalid format for single-sided range
			}
			matchTargetType = "max_range"
			maxMajor, maxMinor, maxPatch, _ = parseSemver(match[2])
		} else { // Exact or prefixed match (`^` or `~`)
			matches := reExactPrefix.FindStringSubmatch(target)
			if matches == nil {
				return "" // Invalid target format: does not match exact, ^, ~, or range
			}

			prefixOp := matches[1]     // Will be "" for exact, or "~", or "^"
			targetVerStr := matches[2] // e.g., "1.2.3"

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
	}

	candidates := [][]int{}
	// Maps cleaned version string to its triplet.
	// We no longer need a map to store original versions as we're returning cleaned ones.
	versionTripletsMap := make(map[string][]int)

	for _, v := range versions { // `versions` now contains cleaned strings
		major, minor, patch, valid := parseSemver(v)
		if !valid {
			continue // Skip invalid available versions
		}
		versionTriplet := []int{major, minor, patch}
		versionTripletsMap[v] = versionTriplet // Store triplet for cleaned version

		// Handle version matching based on matchTargetType
		switch matchTargetType {
		case "exact":
			if major == targetMajor && minor == targetMinor && patch == targetPatch {
				return v // For exact matches, return the cleaned string directly
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
		case "min_range": // >=
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
			if isGreaterThanOrEqualToMin {
				candidates = append(candidates, versionTriplet)
			}
		case "max_range": // <=
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
			if isLessThanOrEqualToMax {
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

	// Convert the top candidate (largest version) back to its cleaned string format
	top := candidates[0]
	for cleanedVer, parts := range versionTripletsMap {
		if parts[0] == top[0] && parts[1] == top[1] && parts[2] == top[2] {
			return cleanedVer // Return the cleaned version string directly
		}
	}

	return ""
}
