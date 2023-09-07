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

// Test for GetBaseRepoAndBranch
func TestGetBaseRepoAndBranch_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)
	mockCmd.On("getRemoteOriginURL", mock.Anything).Return("https://github.com/user/repo.git", nil)
	mockCmd.On("getOriginURL", mock.Anything).Return("https://github.com/origin/repo.git")
	mockCmd.On("getOriginBranch", mock.Anything).Return("main")

	repo, branch := getBaseRepoAndBranch("", "", mockCmd, &realEnvOps{})

	assert.Equal(t, "https://github.com/origin/repo.git", repo)
	assert.Equal(t, "main", branch)
}

func TestGetBaseRepoAndBranch_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	mockCmd.On("gitRootPath", mock.Anything).Return("../", nil)
	mockCmd.On("getOriginURL", mock.Anything).Return("")
	mockCmd.On("getOriginBranch", mock.Anything).Return("")
	repo, branch := getBaseRepoAndBranch("", "", mockCmd, &realEnvOps{})

	assert.Empty(t, repo)
	assert.Empty(t, branch)
}

// Test for GetCurrentPrRepoAndBranch
func TestGetCurrentPrRepoAndBranch_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock the gitRootPath method
	mockCmd.On("gitRootPath", mock.Anything).Return(".", nil)
	mockCmd.On("getCurrentBranch").Return("feature-branch", nil)
	mockCmd.On("getRemoteOriginURL", ".").Return("https://github.com/user/repo.git", nil)

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

func (m *MockCommander) getRemoteOriginURL(repoPath string) (string, error) {
	args := m.Called(repoPath)
	return args.String(0), args.Error(1)
}

func (m *MockCommander) getCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockCommander) getOriginURL(repoPath string) string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCommander) getOriginBranch(repoPath string) string {
	args := m.Called()
	return args.String(0)
}
