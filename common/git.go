// Package common provides utilities for working with Git repositories.
package common

import (
	"errors"
	"fmt"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
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
	// GetCurrentBranch returns the name of the current branch.
	getCurrentBranch() (string, error)
	// GetDefaultBranch returns the name of the default branch.
	getDefaultBranch(repoDir string) (string, error)
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
	remoteURL := strings.TrimSpace(string(output))

	return remoteURL, nil
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
	branch := strings.TrimSpace(string(output))
	if branch == "HEAD" {
		fmt.Println("HEAD means no branch, running in detached mode. This is probable running in GHA")
	}
	return branch, nil
}

func (r *realGitOps) getDefaultBranch(repoPath string) (string, error) {
	// Open the Git repository at the specified path
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}

	// List all references in the repository
	iter, err := repo.References()
	if err != nil {
		return "", err
	}

	// Initialize a variable to store the default branch name
	var defaultBranch string

	// Iterate through references to find the symbolic reference for origin/HEAD
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() && ref.Name().Short() == "origin/HEAD" {
			defaultBranch = strings.TrimPrefix(ref.Target().Short(), "origin/")
			return nil
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if defaultBranch == "" {
		return "", fmt.Errorf("default branch not found")
	}

	return defaultBranch, nil
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

	defaultBranch, err := ops.getDefaultBranch(repo)
	if err != nil {
		return "", "", err
	}

	return remote, defaultBranch, nil
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

	// If environment variable GITHUB_HEAD_REF is set, use that. This is used in GitHub Actions and from a fork.
	baseRef := os.Getenv("GITHUB_HEAD_REF")
	if baseRef != "" {
		branch = baseRef
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

// DetermineAuthMethod determines the appropriate authentication method for a given repository URL.
// The function supports both HTTPS and SSH-based repositories.
//
// For HTTPS repositories:
// - It first checks if the GIT_TOKEN environment variable is set. If so, it uses this as the Personal Access Token (PAT).
// - If the GIT_TOKEN environment variable is not set, no authentication is used for HTTPS repositories.
//
// For SSH repositories:
// - It first checks if the SSH_PRIVATE_KEY environment variable is set. If so, it uses this as the SSH private key.
// - If the SSH_PRIVATE_KEY environment variable is not set, it attempts to use the default SSH key located at ~/.ssh/id_rsa.
// - If neither the environment variable nor the default key is available, no authentication is used for SSH repositories.
//
// Parameters:
// - repoURL: The URL of the Git repository.
//
// Returns:
// - An appropriate AuthMethod based on the repository URL and available credentials.
// - An error if there's an issue parsing the SSH private key or if the private key cannot be cast to an ssh.Signer.
func DetermineAuthMethod(repoURL string) (transport.AuthMethod, error) {
	var pat string
	var sshPrivateKey string
	if strings.HasPrefix(repoURL, "https://") {
		// Check for Personal Access Token (PAT) in environment variable
		envPat, exists := os.LookupEnv("GIT_TOKEN")
		if exists {
			pat = envPat
		}
		if pat != "" {
			return &http.BasicAuth{
				Username: "git", // This can be anything except an empty string
				Password: pat,
			}, nil
		}
	} else if strings.HasPrefix(repoURL, "git@") {
		// SSH authentication
		envSSHKey, exists := os.LookupEnv("SSH_PRIVATE_KEY")
		if exists {
			sshPrivateKey = envSSHKey
		}
		if sshPrivateKey == "" {
			// Attempt to use the default SSH key if none is provided
			defaultKeyPath := os.ExpandEnv("$HOME/.ssh/id_rsa")
			if _, err := os.Stat(defaultKeyPath); !os.IsNotExist(err) {
				// Read the default key
				keyBytes, err := os.ReadFile(defaultKeyPath)
				if err != nil {
					return nil, err
				}
				sshPrivateKey = string(keyBytes)
			}
		}
		if sshPrivateKey != "" {
			key, err := ssh.ParseRawPrivateKey([]byte(sshPrivateKey))
			if err != nil {
				return nil, err
			}
			signer, ok := key.(ssh.Signer)
			if !ok {
				return nil, errors.New("unable to cast private key to ssh.Signer")
			}
			return &gitssh.PublicKeys{User: "git", Signer: signer}, nil
		}
	}
	return nil, nil // No authentication
}
