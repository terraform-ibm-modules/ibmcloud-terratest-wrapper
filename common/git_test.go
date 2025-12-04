package common

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	git "github.com/go-git/go-git/v5"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestGetLatestCommitID(t *testing.T) {
	g := &realGitOps{}
	repoPath := t.TempDir()
	repo, err := g.PlainInit(repoPath)
	require.NoError(t, err)

	// Create a commit
	wt, err := repo.Worktree()
	require.NoError(t, err)
	testFile := "test.txt"
	err = wt.Filesystem.MkdirAll(".", 0755)
	require.NoError(t, err)

	f, err := wt.Filesystem.Create(testFile)
	require.NoError(t, err)
	f.Close()

	_, err = wt.Add(testFile)
	require.NoError(t, err)
	hash, err := wt.Commit("test commit", g.CommitOptions("Test User", "test@example.com"))
	require.NoError(t, err)

	// Test the function
	commitID, err := g.getLatestCommitID(repoPath)

	assert.NoError(t, err)
	assert.Equal(t, hash.String(), commitID)
}

func TestGetLatestCommitID_InvalidRepo(t *testing.T) {
	g := &realGitOps{}
	_, err := g.getLatestCommitID("/nonexistent/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open repo")
}

func TestCommitExistsInRemote_CommitExists(t *testing.T) {
	g := &realGitOps{}
	remoteRepoPath := t.TempDir()
	remoteRepo, err := g.PlainInit(remoteRepoPath)
	require.NoError(t, err)

	// Create a commit in the remote repo
	wt, err := remoteRepo.Worktree()
	require.NoError(t, err)
	f, err := wt.Filesystem.Create("test.txt")
	require.NoError(t, err)
	f.Close()

	_, err = wt.Add("test.txt")
	require.NoError(t, err)

	hash, err := wt.Commit("test commit", g.CommitOptions("Test User", "test@example.com"))
	require.NoError(t, err)

	// Test the function
	exists, err := g.commitExistsInRemote("file://"+remoteRepoPath, hash.String())

	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCommitExistsInRemote_CommitNotExists(t *testing.T) {
	g := &realGitOps{}
	remoteRepoPath := t.TempDir()
	remoteRepo, err := g.PlainInit(remoteRepoPath)
	require.NoError(t, err)

	// Create at least one commit so repo is not empty
	wt, err := remoteRepo.Worktree()
	require.NoError(t, err)
	f, err := wt.Filesystem.Create("test.txt")
	require.NoError(t, err)
	f.Close()

	_, err = wt.Add("test.txt")
	require.NoError(t, err)

	_, err = wt.Commit("initial commit", g.CommitOptions("Test User", "test@example.com"))
	require.NoError(t, err)

	fakeCommitID := "0000000000000000000000000000000000000000"

	exists, err := g.commitExistsInRemote("file://"+remoteRepoPath, fakeCommitID)

	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCommitExistsInRemote_InvalidRemoteURL(t *testing.T) {
	g := &realGitOps{}
	_, err := g.commitExistsInRemote("file:///nonexistent/repo", "0000000000000000000000000000000000000000")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetch failed")
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

// Test ChangesToBePush
func TestChangesToBePush_Positive(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "../", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	// Mock that git log returns some changes
	mockCmd.On("executeCommand", "../", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte("file1.txt\nfile2.txt"), nil)

	changes, files, err := changesToBePush(t, "../", mockCmd)

	assert.NoError(t, err)
	assert.True(t, changes)
	assert.Equal(t, []string{"file1.txt", "file2.txt"}, files)
	mockCmd.AssertExpectations(t)
}

func TestChangesToBePush_Negative(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "../", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	// Mock that git log returns no changes
	mockCmd.On("executeCommand", "../", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte(""), nil)

	changes, files, err := changesToBePush(t, "../", mockCmd)

	assert.NoError(t, err)
	assert.False(t, changes)
	assert.Empty(t, files)
	mockCmd.AssertExpectations(t)
}

func TestChangesToBePush_Error(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "../", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	// Mock that git log command fails
	mockCmd.On("executeCommand", "../", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte(""), errors.New("git command failed"))
	// Mock the fallback git log command
	mockCmd.On("executeCommand", "../", "git", "log", "HEAD", "--name-only", "--pretty=format:", "-n", "1").
		Return([]byte(""), nil)

	_, _, err := changesToBePush(t, "../", mockCmd)

	assert.NoError(t, err)
	mockCmd.AssertExpectations(t)
}

// Test different output variations
func TestChangesToBePush_FormatVariations(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "../", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	// Test with different git log output formats
	mockCmd.On("executeCommand", "../", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte("file1.txt\nfile2.tf\nfile3.md"), nil)

	changes, files, err := changesToBePush(t, "../", mockCmd)

	assert.NoError(t, err)
	assert.True(t, changes)
	assert.Equal(t, []string{"file1.txt", "file2.tf", "file3.md"}, files)
	mockCmd.AssertExpectations(t)
}

// Test complex git output
func TestChangesToBePush_ComplexOutput(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "../", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	// Test with nested paths and empty lines in output
	mockCmd.On("executeCommand", "../", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte("path/to/file1.txt\n\ndeleted-file.go\n\npath/to/nested/file.yaml"), nil)

	changes, files, err := changesToBePush(t, "../", mockCmd)

	assert.NoError(t, err)
	assert.True(t, changes)
	// Since git log returns full paths rather than status codes + files
	assert.ElementsMatch(t, []string{"path/to/file1.txt", "deleted-file.go", "path/to/nested/file.yaml"}, files)
	mockCmd.AssertExpectations(t)
}

// Test with empty paths
func TestChangesToBePush_EmptyPath(t *testing.T) {
	mockCmd := new(MockCommander)
	// Mock that git status returns no uncommitted changes
	mockCmd.On("executeCommand", "", "git", "status", "--porcelain").
		Return([]byte(""), nil)
	mockCmd.On("executeCommand", "", "git", "log", "@{u}..HEAD", "--name-only", "--pretty=format:").
		Return([]byte("file1.txt"), nil)

	changes, files, err := changesToBePush(t, "", mockCmd)

	assert.NoError(t, err)
	assert.True(t, changes)
	assert.Equal(t, []string{"file1.txt"}, files)
	mockCmd.AssertExpectations(t)
}

// Mock functions
type MockCommander struct {
	mock.Mock
}

// CommitOptions implements gitOps.
func (m *MockCommander) CommitOptions(name string, email string) *git.CommitOptions {
	panic("unimplemented")
}

// Init implements gitOps.
func (m *MockCommander) Init(storage *memory.Storage) (*git.Repository, error) {
	panic("unimplemented")
}

// PlainInit implements gitOps.
func (m *MockCommander) PlainInit(remoteRepoPath string) (*git.Repository, error) {
	panic("unimplemented")
}

// PlainOpen implements gitOps.
func (m *MockCommander) PlainOpen(repoPath string) (*git.Repository, error) {
	panic("unimplemented")
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

func (m *MockCommander) getLatestCommitID(repoPath string) (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockCommander) commitExistsInRemote(remoteURL string, commitID string) (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockCommander) executeCommand(dir string, command string, args ...string) ([]byte, error) {
	callArgs := []interface{}{dir, command}
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}
	mockArgs := m.Called(callArgs...)
	return mockArgs.Get(0).([]byte), mockArgs.Error(1)
}

// helper to create a temporary .netrc file
func writeTempNetrc(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	netrcPath := filepath.Join(tmpDir, ".netrc")
	err := os.WriteFile(netrcPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to write temp .netrc: %v", err)
	}
	return netrcPath
}

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"git@github.com:user/repo.git", true},
		{"ssh://git@github.com/user/repo.git", true},
		{"https://github.com/user/repo.git", false},
		{"http://gitlab.com/user/repo.git", false},
	}

	for _, test := range tests {
		got := isSSHURL(test.url)
		if got != test.expected {
			t.Errorf("isSSHURL(%q) = %v; want %v", test.url, got, test.expected)
		}
	}
}

func TestParseNetrcFile(t *testing.T) {
	content := `
machine github.com login user password pass
machine gitlab.com login gluser password glpass
`
	netrcPath := writeTempNetrc(t, content)
	machines, err := parseNetrcFile(netrcPath)
	if err != nil {
		t.Fatalf("parseNetrcFile failed: %v", err)
	}

	if len(machines) != 2 {
		t.Fatalf("expected 2 machines, got %d", len(machines))
	}

	if machines[0].Machine != "github.com" || machines[0].Login != "user" || machines[0].Password != "pass" {
		t.Errorf("unexpected first machine %+v", machines[0].Login)
	}
	if machines[1].Machine != "gitlab.com" || machines[1].Login != "gluser" || machines[1].Password != "glpass" {
		t.Errorf("unexpected second machine %+v", machines[1].Login)
	}
}

func TestLookupNetrcMachine(t *testing.T) {
	machines := []netrcMachine{
		{Machine: "github.com", Login: "user", Password: "pass"},
		{Machine: "gitlab.com", Login: "gluser", Password: "glpass"},
	}

	m := lookupNetrcMachine("https://github.com/repo.git", machines)
	if m == nil || m.Login != "user" {
		t.Errorf("expected github.com login 'user', got %+v", m)
	}

	m = lookupNetrcMachine("https://gitlab.com/repo.git", machines)
	if m == nil || m.Login != "gluser" {
		t.Errorf("expected gitlab.com login 'gluser', got %+v", m)
	}

	m = lookupNetrcMachine("https://bitbucket.org/repo.git", machines)
	if m != nil {
		t.Errorf("expected nil for unknown host, got %+v", m)
	}
}

func TestLoadNetrcAuth(t *testing.T) {
	content := `
machine github.com login user password pass
`
	netrcPath := writeTempNetrc(t, content)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", filepath.Dir(netrcPath))

	auth := loadNetrcAuth("https://github.com/repo.git")
	if auth == nil {
		t.Fatal("expected auth, got nil")
	}

	if auth.Username != "user" || auth.Password != "pass" {
		t.Errorf("unexpected credentials")
	}
}

func TestHttpsAuth_EnvTokenFallback(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmp)

	origGhToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origGhToken)
	os.Setenv("GITHUB_TOKEN", "gh_test_token")

	auth, err := httpsAuth("https://github.com/repo.git")
	if err != nil {
		t.Fatalf("httpsAuth failed: %v", err)
	}

	basicAuth, ok := auth.(*gitHttp.BasicAuth)
	if !ok {
		t.Fatalf("expected BasicAuth, got %T", auth)
	}
	if basicAuth.Password != "gh_test_token" {
		t.Errorf("expected token 'gh_test_token'")
	}
}

func TestGitAutoAuth_HTTPSNetrc(t *testing.T) {
	content := `
machine github.com login user password pass
`
	netrcPath := writeTempNetrc(t, content)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", filepath.Dir(netrcPath))

	auth, err := GitAutoAuth("https://github.com/repo.git")
	if err != nil {
		t.Fatalf("GitAutoAuth failed: %v", err)
	}
	if _, ok := auth.(*gitHttp.BasicAuth); !ok {
		t.Fatalf("expected BasicAuth")
	}
}

func TestGitAutoAuth_AnonymousHTTPS(t *testing.T) {
	auth, err := GitAutoAuth("https://public-repo.org/repo.git")
	if err != nil {
		t.Fatalf("GitAutoAuth failed: %v", err)
	}
	if auth != nil {
		t.Fatalf("expected nil auth for public repo")
	}
}
