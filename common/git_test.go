package common

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// Test for GitRootPath
func TestGitRootPath_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)

	path, err := gitRootPath("../", mockCmd)

	assert.NoError(t, err)
	assert.Equal(t, "../", path)
}

func TestGitRootPath_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("", errors.New("error finding git root"))

	_, err := gitRootPath("../", mockCmd)

	assert.Error(t, err)
}

// Test for GetDefaultRepoAndBranch
func TestGetDefaultRepoAndBranch_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)
	mockCmd.On("getRemoteURL", mock.Anything).Return("https://github.com/user/repo.git", nil)
	mockCmd.On("getDefaultBranch", mock.Anything).Return("main", nil)

	repo, branch, err := getDefaultRepoAndBranch("../", mockCmd)

	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", repo)
	assert.Equal(t, "main", branch)
}

func TestGetDefaultRepoAndBranch_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("", errors.New("error finding git root"))

	_, _, err := getDefaultRepoAndBranch("../", mockCmd)

	assert.Error(t, err)
}

// Test for GetBaseRepoAndBranch
func TestGetBaseRepoAndBranch_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)
	mockCmd.On("getRemoteURL", mock.Anything).Return("https://github.com/user/repo.git", nil)
	mockCmd.On("getSymbolicRef", mock.Anything).Return("refs/remotes/origin/main", nil)

	repo, branch, err := getBaseRepoAndBranch("https://github.com/user/repo.git", "main", mockCmd, &realEnvOps{})

	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", repo)
	assert.Equal(t, "main", branch)
}

func TestGetBaseRepoAndBranch_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("", errors.New("error finding git root"))

	_, _, err := getBaseRepoAndBranch("", "", mockCmd, &realEnvOps{})

	assert.Error(t, err)
}

func TestGetDefaultRepoAndBranch_SSHRemote(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)
	mockCmd.On("getRemoteURL", mock.Anything).Return("git@github.com:terraform-ibm-modules/terraform-ibm-cbr.git", nil)
	mockCmd.On("getDefaultBranch", mock.Anything).Return("main", nil)

	repo, branch, err := getDefaultRepoAndBranch("../", mockCmd)

	assert.NoError(t, err)
	assert.Equal(t, "git@github.com:terraform-ibm-modules/terraform-ibm-cbr.git", repo)
	assert.Equal(t, "main", branch)
}

// Test for GetDefaultRepoAndBranch
func TestGetDefaultRepoAndBranch(t *testing.T) {
	mockCmd := new(MockCommander)

	// Mock the gitRootPath method
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)

	// Mock the getRemoteURL method
	mockCmd.On("getRemoteURL", mock.Anything).Return("https://github.com/user/repo.git", nil)

	// Mock the getSymbolicRef method
	mockCmd.On("getDefaultBranch", mock.Anything).Return("main", nil)

	// Use the mock in the function
	repo, branch, err := getDefaultRepoAndBranch("../", mockCmd)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", repo)
	assert.Equal(t, "main", branch)
}

// Test for GetCurrentPrRepoAndBranch
func TestGetCurrentPrRepoAndBranch_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock the gitRootPath method
	mockCmd.On("getCurrentBranch").Return("feature-branch", nil)
	mockCmd.On("getRemoteURL", ".").Return("https://github.com/user/repo.git", nil)

	repo, branch, err := getCurrentPrRepoAndBranch(mockCmd)

	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", repo)
	assert.Equal(t, "feature-branch", branch)
}

func TestGetCurrentPrRepoAndBranch_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("getCurrentBranch").Return("", errors.New("error finding current branch"))

	_, _, err := getCurrentPrRepoAndBranch(mockCmd)

	assert.Error(t, err)
}

// Mock functions
type MockCommander struct {
	mock.Mock
}

func (m *MockCommander) getDefaultBranch(repoDir string) (string, error) {
	args := m.Called(repoDir)
	return args.String(0), args.Error(1)
}

func (m *MockCommander) gitRootPath(fromPath string) (string, error) {
	args := m.Called(fromPath)
	return args.String(0), args.Error(1)
}

func (m *MockCommander) getRemoteURL(repoPath string) (string, error) {
	args := m.Called(repoPath)
	return args.String(0), args.Error(1)
}

func (m *MockCommander) getCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}
