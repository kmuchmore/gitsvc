package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	fset := flag.NewFlagSet("GitSvc", flag.ExitOnError)
	fset.String("url", "", "URL of the git repository")
	fset.String("token", "", "Token for the git repository")
	fset.String("ssh-key", "", "Path to the SSH key for authentication")
	fset.String("repo-dir", "./repos", "Path to where to clone the repository")

	fset.Parse(os.Args[1:])

	repoURL := fset.Lookup("url").Value.String()
	token := fset.Lookup("token").Value.String()
	sshKeyPath := fset.Lookup("ssh-key").Value.String()
	repoDir := fset.Lookup("repo-dir").Value.String()

	if repoURL == "" {
		log.Fatal("Please provide a URL to the git repository")
	}

	repoName := filepath.Base(repoURL)
	if repoName == "" {
		log.Fatal("Unable to determine the repository name from the URL")
	}
	repoName = strings.TrimSuffix(repoName, filepath.Ext(repoName))
	repoPath := filepath.Join(repoDir, repoName)

	// Initialize the GitService with options
	var serviceOptions []GitServiceOption
	serviceOptions = append(serviceOptions, WithRepoPath(repoPath))
	if token != "" {
		serviceOptions = append(serviceOptions, WithToken(token))
	} else {
		if sshKeyPath == "" {
			// Use default SSH key path
			sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
		}
		// Check if the SSH key exists and use it if it does
		if _, err := os.Stat(sshKeyPath); err == nil {
			serviceOptions = append(serviceOptions, WithSSHKey(sshKeyPath))
		}
	}

	service := NewGitService(serviceOptions...)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := service.CloneRepo(repoURL); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := service.PullRepo(); err != nil {
			log.Println("Failed to update repo, working in offline mode")
		}
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	// Initialize the Handler
	handler := NewHandler(service)

	// Update routes
	e.GET("/file", handler.GetFileHandler)
	e.GET("/update", handler.UpdateRepoHandler)
	e.GET("/tree", handler.GetTreeHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
