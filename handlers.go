package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

// Handler handles HTTP requests
type Handler struct {
	service *GitService
}

// NewHandler creates a new Handler
func NewHandler(service *GitService) *Handler {
	return &Handler{
		service: service,
	}
}

type FileResponse struct {
	Content    interface{} `json:"content" yaml:"content"`
	Timestamp  time.Time   `json:"timestamp" yaml:"timestamp"`
	CommitHash string      `json:"commit_hash" yaml:"commit_hash"`
	Offline    bool        `json:"offline,omitempty" yaml:"offline,omitempty"`
}

type TreeResponse struct {
	Files      []string  `json:"files" yaml:"files"`
	Timestamp  time.Time `json:"timestamp" yaml:"timestamp"`
	CommitHash string    `json:"commit_hash" yaml:"commit_hash"`
	Offline    bool      `json:"offline,omitempty" yaml:"offline,omitempty"`
}

func Yaml(c echo.Context, code int, i interface{}) error {
	c.Response().Status = code
	c.Response().Header().Set(echo.HeaderContentType, "application/yaml")
	return yaml.NewEncoder(c.Response()).Encode(i)
}

func returnResponse(c echo.Context, statusCode int, format string, response interface{}) error {
	if format == "yaml" {
		return Yaml(c, statusCode, response)
	}
	return c.JSON(statusCode, response)
}

// getCommitInfo retrieves the latest commit hash and timestamp
func (h *Handler) getCommitInfo() (string, time.Time, bool) {
	commitHash, err := h.service.GetLatestCommitHash()
	offline := false
	if err != nil {
		offline = true
		commitHash = "unknown"
	}

	timestamp, err := h.service.GetCommitTimestamp()
	if err != nil {
		offline = true
		timestamp = time.Now()
	}

	return commitHash, timestamp, offline
}

// GetFileHandler handles requests to retrieve a file's content
func (h *Handler) GetFileHandler(c echo.Context) error {
	filePath := c.QueryParam("path")
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}

	fullPath := filepath.Join(h.service.RepoPath, filePath)
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	offline := false

	var content interface{}
	if strings.HasSuffix(filePath, ".json") {
		if err := json.Unmarshal(contentBytes, &content); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse JSON"})
		}
	} else if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		if err := yaml.Unmarshal(contentBytes, &content); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse YAML"})
		}
	} else {
		content = string(contentBytes)
	}

	commitHash, timestamp, offline := h.getCommitInfo()

	response := FileResponse{
		Content:    content,
		Timestamp:  timestamp,
		CommitHash: commitHash,
		Offline:    offline,
	}
	if offline {
		response.Offline = true
	}

	return returnResponse(c, http.StatusOK, format, response)
}

// UpdateRepoHandler handles requests to update the repository
func (h *Handler) UpdateRepoHandler(c echo.Context) error {
	// Pull the latest changes from the repository
	err := h.service.PullRepo()
	if err != nil {
		// If pull fails, return offline status with commit info
		commitHash, timestamp, _ := h.getCommitInfo()
		response := map[string]interface{}{
			"message":     "Failed to update repo, working in offline mode",
			"commit_hash": commitHash,
			"timestamp":   timestamp,
		}
		return c.JSON(http.StatusOK, response)
	}

	// After successful pull, get the latest commit info
	commitHash, timestamp, _ := h.getCommitInfo()
	response := map[string]interface{}{
		"message":     "Repo updated",
		"commit_hash": commitHash,
		"timestamp":   timestamp,
	}
	return c.JSON(http.StatusOK, response)
}

// GetTreeHandler handles requests to retrieve the file tree
func (h *Handler) GetTreeHandler(c echo.Context) error {
	var files []string
	format := c.QueryParam("format")
	if format != "yaml" {
		format = "json"
	}
	err := filepath.Walk(h.service.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.Contains(path, ".git") {
				return nil
			}
			relativePath, err := filepath.Rel(h.service.RepoPath, path)
			if err != nil {
				return err
			}
			files = append(files, relativePath)
		}
		return nil
	})
	if err != nil {
		return returnResponse(c, http.StatusInternalServerError, "json", map[string]string{"error": "Failed to build file tree"})
	}

	commitHash, timestamp, offline := h.getCommitInfo()

	response := TreeResponse{
		Files:      files,
		Timestamp:  timestamp,
		CommitHash: commitHash,
		Offline:    offline,
	}

	return returnResponse(c, http.StatusOK, format, response)
}
