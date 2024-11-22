package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func setupTestGitService(t *testing.T) *GitService {
	testRepoPath := filepath.Join(t.TempDir(), "test_repo")
	// Initialize a test GitService
	gs := NewGitService(WithRepoPath(testRepoPath))

	// Initialize git repository
	repo, err := git.PlainInit(gs.RepoPath, false)
	if err != nil {
		panic(err)
	}

	// Create a file
	readmePath := filepath.Join(gs.RepoPath, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo"), 0644)
	if err != nil {
		panic(err)
	}

	// Add and commit the file
	w, err := repo.Worktree()
	if err != nil {
		panic(err)
	}

	_, err = w.Add("README.md")
	if err != nil {
		panic(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		panic(err)
	}

	newGs := NewGitService(WithRepoPath(t.TempDir()))
	newGs.CloneRepo(gs.RepoPath)

	// Add another commit for update testing
	err = os.WriteFile(readmePath, []byte("# Test Repo\nAdded more content"), 0644)
	if err != nil {
		panic(err)
	}

	_, err = w.Add("README.md")
	if err != nil {
		panic(err)
	}

	_, err = w.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		panic(err)
	}

	return newGs
}

func TestGetFileHandler(t *testing.T) {
	// Setup
	e := echo.New()
	service := setupTestGitService(t)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/file?path=README.md&format=json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := handler.GetFileHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "content")
}

func TestUpdateRepoHandler(t *testing.T) {
	// Setup
	e := echo.New()
	service := setupTestGitService(t)
	gs := NewGitService(WithRepoPath(t.TempDir()))
	gs.CloneRepo(service.RepoPath)
	handler := NewHandler(gs)

	req := httptest.NewRequest(http.MethodGet, "/update", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := handler.UpdateRepoHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Repo updated")
}

func TestGetTreeHandler(t *testing.T) {
	// Setup
	e := echo.New()
	service := setupTestGitService(t)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/tree?format=json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := handler.GetTreeHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "files")
}
