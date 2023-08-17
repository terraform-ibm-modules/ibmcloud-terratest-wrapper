// Package common provides utilities for working with Git repositories.
package common

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// gitOps is an interface that abstracts Git operations. This allows for easier testing
// and decoupling of the actual Git commands from the business logic.
type gitOps interface {
	// GitRootPath returns the root directory of the current Git repository.
	gitRootPath(fromPath string) (string, error)
	// GetRemoteURL retrieves the URL of the remote repository.
	getRemoteURL(repoDir string) (string, error)
	// GetSymbolicRef fetches the symbolic reference for the default branch.
	getSymbolicRef(repo string) (string, error)
	// GetCurrentBranch returns the name of the current branch.
	getCurrentBranch() (string, error)
}

// envOps is an interface that abstracts environment variable operations.
// This allows for easier testing and decoupling.
type envOps interface {
	// LookupEnv retrieves the value of the environment variable named by the key.
	lookupEnv(key string) (string, bool)
}

// realGitOps provides the real-world implementation of gitOps, executing actual Git commands.
type realGitOps struct{}

func (r *realGitOps) getRemoteURL(repoDir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *realGitOps) gitRootPath(fromPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = fromPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *realGitOps) getSymbolicRef(repo string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *realGitOps) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine the PR branch: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// realEnvOps provides the real-world implementation of envOps, interacting with actual environment variables.
type realEnvOps struct{}

func (e *realEnvOps) lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// GitRootPath returns the root directory of the current Git repository.
//
// Parameters:
// - fromPath: The directory from which the Git command should be executed.
//
// Returns:
// - A string representing the path to the root directory of the Git repository.
// - An error if the command fails or the directory is not part of a Git repository.
func GitRootPath(fromPath string) (string, error) {
	return gitRootPath(fromPath, &realGitOps{})
}

func gitRootPath(fromPath string, ops gitOps) (string, error) {
	return ops.gitRootPath(fromPath)
}

// GetDefaultRepoAndBranch determines the default repository URL and branch name
// of the current Git repository. This function is useful when you want to programmatically
// determine the repository and branch details without relying on manual input.
//
// Parameters:
//   - path: The directory from which the Git commands should be executed. This is typically
//     the root directory of your project or any sub-directory within it.
//
// Returns:
// - A string representing the default repository URL without any credentials.
// - A string representing the default branch name.
// - An error if any of the Git commands fail or if the repository/branch details cannot be determined.
func GetDefaultRepoAndBranch(path string) (string, string, error) {
	return getDefaultRepoAndBranch(path, &realGitOps{})
}

func getDefaultRepoAndBranch(path string, ops gitOps) (string, string, error) {
	repo, err := ops.gitRootPath(path)
	if err != nil {
		return "", "", err
	}

	remote, err := ops.getRemoteURL(repo)
	if err != nil {
		return "", "", err
	}

	parsedURL, err := url.Parse(remote)
	if err != nil {
		return "", "", fmt.Errorf("error parsing URL: %v", err)
	}
	parsedURL.User = nil
	defaultRepo := parsedURL.String()

	branchRef, err := ops.getSymbolicRef(repo)
	if err != nil {
		return "", "", err
	}
	defaultBranch := strings.TrimSpace(branchRef)
	defaultBranch = strings.TrimPrefix(defaultBranch, "refs/remotes/origin/")

	return defaultRepo, defaultBranch, nil
}

// GetBaseRepoAndBranch determines the base repository URL and branch name based on a hierarchy of sources.
// The function first checks the provided arguments, then checks environment variables, and finally,
// if neither source provides the values, it uses Git logic to fetch the details.
// This function is useful in scenarios where you want to determine the base repository and branch
// details from multiple potential sources.
//
// Parameters:
//   - repo: The initial repository URL to consider. This can be an empty string if you want the function
//     to determine the repository URL from other sources.
//   - branch: The initial branch name to consider. This can be an empty string if you want the function
//     to determine the branch name from other sources.
//
// Returns:
// - A string representing the base repository URL.
// - A string representing the base branch name.
// - An error if any of the Git commands fail or if the repository/branch details cannot be determined.
func GetBaseRepoAndBranch(repo string, branch string) (string, string, error) {
	return getBaseRepoAndBranch(repo, branch, &realGitOps{}, &realEnvOps{})
}

func getBaseRepoAndBranch(repo string, branch string, git gitOps, env envOps) (string, string, error) {
	envRepo, exists := env.lookupEnv("BASE_TERRAFORM_REPO")
	if exists {
		repo = envRepo
	}
	envBranch, exists := env.lookupEnv("BASE_TERRAFORM_BRANCH")
	if exists {
		branch = envBranch
	}

	if repo == "" || branch == "" {
		defaultRepo, defaultBranch, err := getDefaultRepoAndBranch("../", git)
		if err != nil {
			return "", "", err
		}
		if repo == "" {
			repo = defaultRepo
		}
		if branch == "" {
			branch = defaultBranch
		}
	}

	return repo, branch, nil
}

// GetCurrentPrRepoAndBranch returns the repository URL and branch name of the current PR.
//
// Returns:
// - A string representing the repository URL of the current PR.
// - A string representing the branch name of the current PR.
// - An error if any of the Git commands fail or if the repository/branch details cannot be determined.
func GetCurrentPrRepoAndBranch() (string, string, error) {
	return getCurrentPrRepoAndBranch(&realGitOps{})
}

func getCurrentPrRepoAndBranch(ops gitOps) (string, string, error) {
	// Get the current branch name
	branch, err := ops.getCurrentBranch()
	if err != nil {
		return "", "", err
	}

	// Get the remote URL for the current branch
	repoURL, err := ops.getRemoteURL(".")
	if err != nil {
		return "", "", err
	}

	return repoURL, branch, nil
}
