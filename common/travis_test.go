package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type travisVariables struct {
	buildNumber string
	buildId     string
	pullRequest string
	repoSlug    string
}

func getTravisVars() travisVariables {
	var t travisVariables

	t.buildNumber = "1234"
	t.buildId = "1"
	t.pullRequest = "12"
	t.repoSlug = "Testing/test"

	return t
}

func TestGetTagsInTravis(t *testing.T) {
	t.Setenv("TRAVIS_BUILD_NUMBER", getTravisVars().buildNumber)
	t.Setenv("TRAVIS_BUILD_ID", getTravisVars().buildId)
	t.Setenv("TRAVIS_PULL_REQUEST", getTravisVars().pullRequest)
	t.Setenv("TRAVIS_REPO_SLUG", getTravisVars().repoSlug)
	expected := []string{"travis-build-" + getTravisVars().buildNumber,
		"travis-build-id-" + getTravisVars().buildId,
		"PR-" + getTravisVars().pullRequest,
		strings.ReplaceAll(getTravisVars().repoSlug, "/", "-"),
	}

	assert.Equal(t, expected, GetTagsFromTravis())
}

func TestGetTagsInTravisPrMissing(t *testing.T) {
	t.Setenv("TRAVIS_BUILD_NUMBER", getTravisVars().buildNumber)
	t.Setenv("TRAVIS_BUILD_ID", getTravisVars().buildId)
	t.Setenv("TRAVIS_REPO_SLUG", getTravisVars().repoSlug)
	expected := []string{"travis-build-" + getTravisVars().buildNumber,
		"travis-build-id-" + getTravisVars().buildId,
		strings.ReplaceAll(getTravisVars().repoSlug, "/", "-"),
	}
	assert.Equal(t, expected, GetTagsFromTravis())
}

func TestGetTagsInTravisPrRepo(t *testing.T) {
	t.Setenv("TRAVIS_BUILD_NUMBER", getTravisVars().buildNumber)
	t.Setenv("TRAVIS_BUILD_ID", getTravisVars().buildId)
	t.Setenv("TRAVIS_PULL_REQUEST", getTravisVars().pullRequest)
	expected := []string{"travis-build-" + getTravisVars().buildNumber,
		"travis-build-id-" + getTravisVars().buildId,
		"PR-" + getTravisVars().pullRequest,
	}
	assert.Equal(t, expected, GetTagsFromTravis())
}

func TestGetTagsOutsideTravis(t *testing.T) {
	var expected []string

	assert.Equal(t, expected, GetTagsFromTravis())
}
