package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type GitService struct {
	RepoPath   string
	Repo       *git.Repository
	Token      string
	SSHKeyPath string
}

type GitServiceOption func(*GitService)

func WithRepoPath(repoPath string) GitServiceOption {
	return func(gs *GitService) {
		gs.RepoPath = repoPath
	}
}

func WithToken(token string) GitServiceOption {
	return func(gs *GitService) {
		gs.Token = token
	}
}

func WithSSHKey(sshKeyPath string) GitServiceOption {
	return func(gs *GitService) {
		gs.SSHKeyPath = sshKeyPath
	}
}

func NewGitService(opts ...GitServiceOption) *GitService {
	gs := &GitService{}
	for _, opt := range opts {
		opt(gs)
	}
	return gs
}

func (s *GitService) getAuth() (transport.AuthMethod, error) {
	if s.Token != "" {
		return &http.BasicAuth{
			Username: "token", // can be anything except an empty string
			Password: s.Token,
		}, nil
	} else if s.SSHKeyPath != "" {
		return ssh.NewPublicKeysFromFile("git", s.SSHKeyPath, "")
	} else {
		// Use default SSH key path
		defaultKeyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
		if _, err := os.Stat(defaultKeyPath); err == nil {
			return ssh.NewPublicKeysFromFile("git", defaultKeyPath, "")
		}
	}
	return nil, nil
}

func (s *GitService) CloneRepo(repoURL string) error {
	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	}
	auth, err := s.getAuth()
	if err != nil {
		return err
	}
	if auth != nil {
		cloneOptions.Auth = auth
	}
	repo, err := git.PlainClone(s.RepoPath, false, cloneOptions)
	if err != nil {
		return err
	}
	s.Repo = repo
	return nil
}

func (s *GitService) PullRepo() error {
	if s.Repo == nil {
		repo, err := git.PlainOpen(s.RepoPath)
		if err != nil {
			return err
		}
		s.Repo = repo
	}

	w, err := s.Repo.Worktree()
	if err != nil {
		return err
	}

	pullOptions := &git.PullOptions{
		RemoteName:        "origin",
		Progress:          os.Stdout,
		SingleBranch:      false,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}
	auth, err := s.getAuth()
	if err != nil {
		return err
	}
	if auth != nil {
		pullOptions.Auth = auth
	}

	err = w.Pull(pullOptions)
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}

func (s *GitService) GetLatestCommitHash() (string, error) {
	if s.Repo == nil {
		repo, err := git.PlainOpen(s.RepoPath)
		if err != nil {
			return "", err
		}
		s.Repo = repo
	}

	ref, err := s.Repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

func (s *GitService) GetCommitTimestamp() (time.Time, error) {
	if s.Repo == nil {
		repo, err := git.PlainOpen(s.RepoPath)
		if err != nil {
			return time.Time{}, err
		}
		s.Repo = repo
	}

	ref, err := s.Repo.Head()
	if err != nil {
		return time.Time{}, err
	}

	commit, err := s.Repo.CommitObject(ref.Hash())
	if err != nil {
		return time.Time{}, err
	}

	return commit.Committer.When, nil
}
