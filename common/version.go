package common

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// parse function is used by MatchVersion function to find the most suitable version
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

// Main matching function
// This function takes all the versions avialable for a dependency
// and returns the suitable version matching target
// here target can be an actual version or unpinned version like ^v3.0.1 or ~v4.1.4
func MatchVersion(versions []string, target string) string {
	operator := ""
	if strings.HasPrefix(target, "^") || strings.HasPrefix(target, "~") {
		operator = string(target[0])
		target = target[1:]
	}

	targetMajor, targetMinor, targetPatch, ok := parseSemver(target)
	if !ok {
		return ""
	}

	candidates := [][]int{}
	versionMap := map[string][]int{}

	for _, v := range versions {
		major, minor, patch, valid := parseSemver(v)
		if !valid {
			continue
		}
		versionTriplet := []int{major, minor, patch}
		versionMap[v] = versionTriplet

		// Handle version matching based on operator
		switch operator {
		case "^":
			if major == targetMajor {
				candidates = append(candidates, versionTriplet)
			}
		case "~":
			if major == targetMajor && minor == targetMinor {
				candidates = append(candidates, versionTriplet)
			}
		default:
			// Exact match
			if major == targetMajor && minor == targetMinor && patch == targetPatch {
				return v
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by major, minor, patch descending
	sort.SliceStable(candidates, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if candidates[i][k] != candidates[j][k] {
				return candidates[i][k] > candidates[j][k]
			}
		}
		return false
	})

	// Convert top candidate back to string and find original version string
	top := candidates[0]
	for ver, parts := range versionMap {
		if parts[0] == top[0] && parts[1] == top[1] && parts[2] == top[2] {
			return ver
		}
	}

	return ""
}
