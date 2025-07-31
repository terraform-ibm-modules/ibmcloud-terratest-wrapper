// Package common provides utilities for working with Git repositories.
package common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"golang.org/x/crypto/ssh"
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
	// ExecuteCommand executes a command and returns its output.
	executeCommand(dir string, command string, args ...string) ([]byte, error)
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
		log.Println("Unable to determine Git root path")
		log.Printf("Checking if the fromPath: %s directory is the root\n", fromPath)
		// if fromPath contains .git, then it is the root
		// otherwise, return the error
		if _, errNotExist := os.Stat(fromPath + "/.git"); os.IsNotExist(errNotExist) {
			// check if the current working directory is the root
			cwd, errGetCwd := os.Getwd()
			if errGetCwd != nil {
				return "", fmt.Errorf("failed to determine the Git root path: %s %v", output, err)
			} else {
				log.Println("fromPath is not the root")
				log.Printf("Checking if the current working directory: %s is the root\n", cwd)
				if _, errNotExist := os.Stat(cwd + "/.git"); os.IsNotExist(errNotExist) {
					log.Println("current working directory is not the root")
					return "", fmt.Errorf("failed to determine the Git root path: %s %v", output, err)
				}
				log.Println("current working directory is the root")
				return cwd, nil
			}
		}
		log.Println("fromPath is the root")
		return fromPath, nil
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
		fmt.Println("HEAD means no branch, running in detached mode. This is probably running in GHA")
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

// Implementation of executeCommand for realGitOps
func (r *realGitOps) executeCommand(dir string, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	return cmd.Output()
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
			key, err := RetrievePrivateKey(sshPrivateKey)
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

// RetrievePrivateKey is a function that takes a string sshPvtKey as input and returns an interface{} and error as output.
// IF the SSH_PASSPHRASE environment variable is set:
//  - It will parse the raw private key with passphrase using the ParseRawPrivateKeyWithPassphrase method of the ssh package.
// IF the SSH_PASSPHRASE environment variable is NOT set or an empty string:
//  - It will parse the raw private key without passphrase using the ParseRawPrivateKey method of the ssh package.
// In both cases:
// - If an error is returned, then return nil and error.
// - Otherwise return the parsed key as interface{} and nil.

// Parameters:
// sshPvtKey: The raw ssh private key.

// Returns:
// - An interface{} that contains the parsed private key.
// - An error (if any)

func RetrievePrivateKey(sshPvtKey string) (interface{}, error) {
	var sshPassphrase string
	// Chek for SSH_PASSPHRASE environment variable
	envSSHPassphrase, isPassphrase := os.LookupEnv("SSH_PASSPHRASE")
	if isPassphrase {
		sshPassphrase = envSSHPassphrase
	}
	if sshPassphrase != "" {
		key, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(sshPvtKey), []byte(sshPassphrase))
		if err != nil {
			return nil, err
		}
		return key, err
	}
	// Use method without SSH PASSPHRASE IF NOT PROVIDED
	key, err := ssh.ParseRawPrivateKey([]byte(sshPvtKey))
	if err != nil {
		return nil, err
	}
	return key, err
}

// SkipUpgradeTest can determine if a terraform or schematics upgrade test should be skipped by analyzing
// the currently checked out git branch, looking for specific verbage in the commit messages.
func SkipUpgradeTest(testing *testing.T, source_repo string, source_branch string, branch string) bool {

	// random string to use in remote name
	remote := fmt.Sprintf("upstream-%s", strings.ToLower(random.UniqueId()))
	logger.Log(testing, "Remote name:", remote)
	// Set upstream to the source repo
	remote_out, remote_err := exec.Command("/bin/sh", "-c", fmt.Sprintf("git remote add %s %s", remote, source_repo)).Output()
	if remote_err != nil {
		logger.Log(testing, "Add remote output:\n", remote_out)
		logger.Log(testing, "Error adding upstream remote:\n", remote_err)
		return false
	}
	// Fetch the source repo
	fetch_out, fetch_err := exec.Command("/bin/sh", "-c", fmt.Sprintf("git fetch %s -f", remote)).Output()
	if fetch_err != nil {
		logger.Log(testing, "Fetch output:\n", fetch_out)
		logger.Log(testing, "Error fetching upstream:\n", fetch_err)
		return false
	} else {
		logger.Log(testing, "Fetch output:\n", fetch_out)
	}
	// Get all the commit messages from the PR branch
	// NOTE: using the "origin" of the default branch as the start point, which will exist in a fresh
	// clone even if the default branch has not been checked out or pulled.
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("git log %s/%s..%s", remote, source_branch, branch))
	out, _ := cmd.CombinedOutput()

	fmt.Printf("Commit Messages (%s): \n%s", source_branch, string(out))
	// Skip upgrade Test if BREAKING CHANGE OR SKIP UPGRADE TEST string found in commit messages
	doNotRunUpgradeTest := false
	if (strings.Contains(string(out), "BREAKING CHANGE") || strings.Contains(string(out), "SKIP UPGRADE TEST")) && !strings.Contains(string(out), "UNSKIP UPGRADE TEST") {
		doNotRunUpgradeTest = true
	}

	return doNotRunUpgradeTest
}

func CloneAndCheckoutBranch(testing *testing.T, repoURL string, branch string, cloneDir string) error {

	authMethod, authErr := DetermineAuthMethod(repoURL)
	if authErr != nil {
		logger.Log(testing, "Failed to determine authentication method, trying without authentication...")

		// Convert SSH URL to HTTPS URL
		if strings.HasPrefix(repoURL, "git@") {
			repoURL = strings.Replace(repoURL, ":", "/", 1)
			repoURL = strings.Replace(repoURL, "git@", "https://", 1)
			repoURL = strings.TrimSuffix(repoURL, ".git") + ".git"
		}

		// Try to clone without authentication
		_, errUnauth := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:           repoURL,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			SingleBranch:  true,
		})

		if errUnauth != nil {
			// If unauthenticated clone fails and we cannot determine authentication, return the error from the unauthenticated approach
			return fmt.Errorf("failed to determine authentication method and clone base repo and branch without authentication: %v", errUnauth)
		} else {
			logger.Log(testing, "Cloned base repo and branch without authentication")
		}
	} else {
		// Authentication method determined, try with authentication
		_, errAuth := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:           repoURL,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			SingleBranch:  true,
			Auth:          authMethod,
		})

		if errAuth != nil {
			logger.Log(testing, "Failed to clone base repo and branch with authentication, trying without authentication...")
			// Convert SSH URL to HTTPS URL
			if strings.HasPrefix(repoURL, "git@") {
				repoURL = strings.Replace(repoURL, ":", "/", 1)
				repoURL = strings.Replace(repoURL, "git@", "https://", 1)
				repoURL = strings.TrimSuffix(repoURL, ".git") + ".git"
			}

			// Try to clone without authentication
			_, errUnauth := git.PlainClone(cloneDir, false, &git.CloneOptions{
				URL:           repoURL,
				ReferenceName: plumbing.NewBranchReferenceName(branch),
				SingleBranch:  true,
			})

			if errUnauth != nil {
				// If unauthenticated clone also fails, return the error from the authenticated approach
				return fmt.Errorf("failed to clone base repo and branch with authentication: %v", errAuth)
			} else {
				logger.Log(testing, "Cloned base repo and branch without authentication")
			}
		} else {
			logger.Log(testing, "Cloned base repo and branch with authentication")
		}
	}

	return nil
}

// ChangesToBePush determines if there are any changes to push to the remote repository.
// Returns a boolean indicating if there are changes and a slice of filenames that have changes.
func ChangesToBePush(testing *testing.T, repoDir string) (bool, []string, error) {
	return changesToBePush(testing, repoDir, &realGitOps{})
}

func changesToBePush(testing *testing.T, repoDir string, git gitOps) (bool, []string, error) {
	// Check for uncommitted changes
	uncommittedOutput, uncommittedErr := git.executeCommand(repoDir, "git", "status", "--porcelain")
	if uncommittedErr != nil {
		logger.Log(testing, "Failed to determine if there are uncommitted changes:", uncommittedErr)
		return false, nil, uncommittedErr
	}

	// Check for unpushed commits
	unpushedOutput, unpushedErr := git.executeCommand(repoDir, "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:")

	// If there's an error, it might be because there's no upstream branch
	// In that case, let's try to check if there are any commits at all
	if unpushedErr != nil {
		logger.Log(testing, "Failed to check unpushed commits, trying alternative approach:", unpushedErr)
		// Check if there are any commits
		unpushedOutput, unpushedErr = git.executeCommand(repoDir, "git", "log", "HEAD", "--name-only", "--pretty=format:", "-n", "1")
		if unpushedErr != nil {
			logger.Log(testing, "Failed to determine if there are commits:", unpushedErr)
			// Continue with just the uncommitted changes check
			unpushedOutput = []byte{}
		}
	}

	// Process uncommitted changes
	hasUncommittedChanges := len(uncommittedOutput) > 0
	uncommittedFiles := make([]string, 0)

	if hasUncommittedChanges {
		// Parse output to extract filenames for uncommitted changes
		lines := strings.Split(strings.TrimSpace(string(uncommittedOutput)), "\n")

		for _, line := range lines {
			if len(line) > 0 {
				// git status --porcelain output has the format: XY filename
				// where X and Y are status codes and the rest is the filename
				parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
				if len(parts) > 1 {
					// There might be multiple spaces between status and filename
					filename := strings.TrimSpace(parts[1])
					uncommittedFiles = append(uncommittedFiles, filename)
				} else if len(parts) == 1 && len(parts[0]) > 2 {
					// Handle cases where there's no space after status codes
					uncommittedFiles = append(uncommittedFiles, strings.TrimSpace(parts[0][2:]))
				}
			}
		}
	}

	// Process unpushed commits
	hasUnpushedCommits := len(unpushedOutput) > 0
	unpushedFiles := make([]string, 0)

	if hasUnpushedCommits && string(unpushedOutput) != "" {
		// Parse output to extract filenames for unpushed commits
		lines := strings.Split(strings.TrimSpace(string(unpushedOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				unpushedFiles = append(unpushedFiles, line)
			}
		}
	}

	// Combine both lists of files and remove duplicates
	allChangedFiles := make([]string, 0, len(uncommittedFiles)+len(unpushedFiles))
	allChangedFiles = append(allChangedFiles, uncommittedFiles...)

	// Add unpushed files, avoiding duplicates
	fileMap := make(map[string]bool)
	for _, file := range uncommittedFiles {
		fileMap[file] = true
	}

	for _, file := range unpushedFiles {
		if !fileMap[file] {
			allChangedFiles = append(allChangedFiles, file)
			fileMap[file] = true
		}
	}

	return hasUncommittedChanges || hasUnpushedCommits, allChangedFiles, nil
}

// CheckRemoteBranchExists checks if a specific branch exists in a remote Git repository
// repoURL: the HTTPS URL of the repository (e.g., "https://github.com/user/repo")
// branchName: the name of the branch to check (e.g., "main", "feature-branch")
// Returns true if the branch exists, false otherwise, and an error if the check fails
func CheckRemoteBranchExists(repoURL, branchName string) (bool, error) {
	if repoURL == "" || branchName == "" {
		return false, fmt.Errorf("repository URL and branch name must not be empty")
	}

	// Use git ls-remote to check if the branch exists without cloning the repo
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL, branchName)
	output, err := cmd.Output()
	if err != nil {
		// Check if it's a repository access error
		if exitError, ok := err.(*exec.ExitError); ok {
			return false, fmt.Errorf("failed to access repository '%s': %s", repoURL, string(exitError.Stderr))
		}
		return false, fmt.Errorf("failed to check remote branch: %w", err)
	}

	// If output is empty, the branch doesn't exist
	// If output has content, the branch exists (git ls-remote returns "commit_hash refs/heads/branch_name")
	result := strings.TrimSpace(string(output))
	return result != "", nil
}

// GetFileDiff returns the git diff output for a specific file
// repoDir: the directory containing the git repository
// fileName: the name of the file to get the diff for
// Returns the diff output as a string and any error encountered
func GetFileDiff(repoDir string, fileName string) (string, error) {
	return getFileDiff(repoDir, fileName, &realGitOps{})
}

func getFileDiff(repoDir string, fileName string, git gitOps) (string, error) {
	// Get the diff for the specific file
	diffOutput, err := git.executeCommand(repoDir, "git", "diff", fileName)
	if err != nil {
		return "", fmt.Errorf("failed to get diff for file %s: %w", fileName, err)
	}

	// If there's no staged diff, try to get unstaged diff
	if len(diffOutput) == 0 {
		diffOutput, err = git.executeCommand(repoDir, "git", "diff", "--cached", fileName)
		if err != nil {
			return "", fmt.Errorf("failed to get cached diff for file %s: %w", fileName, err)
		}
	}

	return string(diffOutput), nil
}
