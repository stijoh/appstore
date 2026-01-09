/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chartsync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Syncer handles periodic synchronization of Helm charts from a Git repository
type Syncer struct {
	repoURL      string
	branch       string
	localPath    string
	syncInterval time.Duration
	repo         *git.Repository
	mu           sync.RWMutex
	logger       logr.Logger
}

// NewSyncer creates a new chart syncer
func NewSyncer(repoURL, branch, localPath string, syncInterval time.Duration) *Syncer {
	return &Syncer{
		repoURL:      repoURL,
		branch:       branch,
		localPath:    localPath,
		syncInterval: syncInterval,
		logger:       ctrl.Log.WithName("chartsync"),
	}
}

// Start begins the periodic sync process
func (s *Syncer) Start(ctx context.Context) error {
	// Initial clone or open
	if err := s.initialSync(); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Start periodic sync
	go s.periodicSync(ctx)

	return nil
}

// initialSync clones the repo or opens existing one
func (s *Syncer) initialSync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting initial chart sync", "repo", s.repoURL, "path", s.localPath)

	// Check if repo already exists locally
	if _, err := os.Stat(filepath.Join(s.localPath, ".git")); err == nil {
		// Open existing repo
		repo, err := git.PlainOpen(s.localPath)
		if err != nil {
			s.logger.Error(err, "Failed to open existing repo, will re-clone")
			os.RemoveAll(s.localPath)
		} else {
			s.repo = repo
			// Pull latest changes
			if err := s.pull(); err != nil {
				s.logger.Error(err, "Failed to pull, will re-clone")
				os.RemoveAll(s.localPath)
				s.repo = nil
			} else {
				s.logger.Info("Opened existing repo and pulled latest changes")
				return nil
			}
		}
	}

	// Clone fresh
	s.logger.Info("Cloning charts repository")
	repo, err := git.PlainClone(s.localPath, false, &git.CloneOptions{
		URL:           s.repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(s.branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}

	s.repo = repo
	s.logger.Info("Charts repository cloned successfully")
	return nil
}

// pull fetches and merges latest changes
func (s *Syncer) pull() error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	w, err := s.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(s.branch),
		SingleBranch:  true,
		Force:         true,
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}

// periodicSync runs sync on interval
func (s *Syncer) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping periodic chart sync")
			return
		case <-ticker.C:
			s.mu.Lock()
			if err := s.pull(); err != nil {
				s.logger.Error(err, "Periodic sync failed")
			} else {
				s.logger.V(1).Info("Periodic sync completed")
			}
			s.mu.Unlock()
		}
	}
}

// GetChartPath returns the local path to a chart
func (s *Syncer) GetChartPath(chartName string) string {
	return filepath.Join(s.localPath, chartName)
}

// ChartExists checks if a chart exists locally
func (s *Syncer) ChartExists(chartName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chartPath := s.GetChartPath(chartName)
	chartYaml := filepath.Join(chartPath, "Chart.yaml")

	_, err := os.Stat(chartYaml)
	return err == nil
}

// ListCharts returns all available charts
func (s *Syncer) ListCharts() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read charts directory: %w", err)
	}

	var charts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if entry.Name()[0] == '.' {
			continue
		}
		// Check if it's a valid chart (has Chart.yaml)
		chartYaml := filepath.Join(s.localPath, entry.Name(), "Chart.yaml")
		if _, err := os.Stat(chartYaml); err == nil {
			charts = append(charts, entry.Name())
		}
	}

	return charts, nil
}

// ForceSync triggers an immediate sync
func (s *Syncer) ForceSync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Force sync triggered")
	return s.pull()
}
