// Package common provides utilities for working with Git repositories.
package common

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// gitOps is an interface that abstracts Git operations. This allows for easier testing
// and decoupling of the actual Git commands from the business logic.
type gitOps interface {
	// GitRootPath returns the root directory of the current Git repository.
	gitRootPath(fromPath string) (string, error)
	// GetRemoteURL retrieves the URL of the remote repository.
	getRemoteOriginURL(repoDir string) (string, error)
	// GetCurrentBranch returns the name of the current branch.
	getCurrentBranch() (string, error)
	// GetOriginURL returns the URL of the origin repository.
	getOriginURL(repoPath string) string
	// GetOriginBranch returns the name of the origin branch.
	getOriginBranch(repoPath string) string
}

// envOps is an interface that abstracts environment variable operations.
// This allows for easier testing and decoupling.
type envOps interface {
	// LookupEnv retrieves the value of the environment variable named by the key.
	lookupEnv(key string) (string, bool)
}

// realGitOps provides the real-world implementation of gitOps, executing actual Git commands.
type realGitOps struct{}

func (r *realGitOps) getRemoteOriginURL(repoDir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine the remote origin URL: %s %v", output, err)
	}
	remoteURL := strings.TrimSpace(string(output))

	return remoteURL, nil
}

func (r *realGitOps) gitRootPath(fromPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = fromPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine the Git root path: %s %v", output, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *realGitOps) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	output, _ := cmd.Output()
	// If the output is empty, try to get the branch name using git rev-parse
	if string(output) == "" {
		cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		output, _ = cmd.Output()
	}
	// If the output is still empty, try to get the branch name using git status
	if string(output) == "" {
		cmd := exec.Command("git", "status", "--branch", "--porcelain")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to determine the current branch: %s %v", output, err)
		}

		// Parse the output to extract the current branch name.
		re := regexp.MustCompile(`## (.+)\.\.\.`)
		matches := re.FindStringSubmatch(string(output))
		if len(matches) != 2 {
			return "", fmt.Errorf("failed to determine the current branch: unable to parse git status")
		}

	}
	branch := strings.TrimSpace(string(output))
	if branch == "HEAD" {
		fmt.Println("HEAD means no branch, running in detached mode. This is probable running in GHA")
	}
	return branch, nil
}

func (r *realGitOps) getOriginURL(repoPath string) string {
	// Determine the URL of the upstream remote (usually "origin")
	repo := ""
	cmd := exec.Command("git", "remote", "get-url", "upstream")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil { // Check if the first command is successful
		repo = strings.TrimSpace(string(output))
	} else {
		// If there's no "upstream" remote, fall back to "origin"
		cmd := exec.Command("git", "remote", "get-url", "origin")
		output, err = cmd.Output()
		if err != nil {
			log.Println("Unable to determine origin URL")
			log.Println(err)
			return ""
		}
		repo = strings.TrimSpace(string(output))
	}

	return repo
}

func (r *realGitOps) getOriginBranch(repoPath string) string {
	branch := ""
	// Try to get the branch from the "origin" remote
	cmd := exec.Command("git", "remote", "show", "origin")
	output, err := cmd.Output()
	if err == nil { // Check if the first command is successful
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "HEAD branch:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					branch = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// If branch is still empty, try to get it from the "upstream" remote
	if branch == "" {
		cmd := exec.Command("git", "remote", "show", "upstream")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "HEAD branch:") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						branch = strings.TrimSpace(parts[1])
						break
					}
				}
			}
		}
	}

	// If branch is still empty, use an alternative method to get the current branch
	if branch == "" {
		cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
		output, err := cmd.Output()
		if err == nil {
			branch = strings.TrimSpace(string(output))
		}
	}

	return branch
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
func GetBaseRepoAndBranch(repo string, branch string) (string, string) {
	return getBaseRepoAndBranch(repo, branch, &realGitOps{}, &realEnvOps{})
}

func getBaseRepoAndBranch(repo string, branch string, git gitOps, env envOps) (string, string) {
	envRepo, exists := env.lookupEnv("BASE_TERRAFORM_REPO")
	if exists {
		repo = envRepo
	}
	envBranch, exists := env.lookupEnv("BASE_TERRAFORM_BRANCH")
	if exists {
		branch = envBranch
	}

	if repo == "" || branch == "" {
		repoPath, err := git.gitRootPath(".")
		if err != nil {
			log.Fatal(err)
		}
		repo = git.getOriginURL(repoPath)
		branch = git.getOriginBranch(repoPath)
	}

	return repo, branch
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

func getCurrentPrRepoAndBranch(git gitOps) (string, string, error) {
	// Get the current branch name
	branch, err := git.getCurrentBranch()
	if err != nil {
		return "", "", err
	}

	repoPath, err := git.gitRootPath(".")
	if err != nil {
		return "", "", err
	}
	// Get the remote URL for the current branch
	repoURL, err := git.getRemoteOriginURL(repoPath)
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
