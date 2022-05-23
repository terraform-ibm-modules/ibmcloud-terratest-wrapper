package testhelper

import (
	"os"
	"strings"
)

// GetTagsFromTravis Generates a list of tags to add to resources if running in Travis.
// Returns empty list if not in Travis
func GetTagsFromTravis() []string {
	// List of tags to add to created resources
	var tags []string

	// If running in Travis add tags
	travisBuild, inTravis := os.LookupEnv("TRAVIS_BUILD_NUMBER")
	if inTravis {
		travisBuildId, _ := os.LookupEnv("TRAVIS_BUILD_ID")
		tags = append(tags, "travis-build-"+travisBuild)
		tags = append(tags, "travis-build-id-"+travisBuildId)

		prNumber, prExists := os.LookupEnv("TRAVIS_PULL_REQUEST")
		repo, repoExists := os.LookupEnv("TRAVIS_REPO_SLUG")
		if prExists {
			tags = append(tags, "PR-"+prNumber)
		}
		if repoExists {
			repo = strings.ReplaceAll(repo, "/", "-")
			tags = append(tags, repo)
		}
	}
	return tags
}
