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

package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client wraps Helm SDK operations
type Client struct {
	settings   *cli.EnvSettings
	chartsPath string
	repoURL    string
	mu         sync.Mutex
}

// ReleaseInfo contains information about a Helm release
type ReleaseInfo struct {
	Name         string
	Namespace    string
	Revision     int
	Status       string
	ChartName    string
	ChartVersion string
	AppVersion   string
	Updated      time.Time
}

// NewClient creates a new Helm client
func NewClient(chartsPath, repoURL string) *Client {
	settings := cli.New()
	return &Client{
		settings:   settings,
		chartsPath: chartsPath,
		repoURL:    repoURL,
	}
}

// getActionConfig creates a Helm action configuration for the given namespace
func (c *Client) getActionConfig(ctx context.Context, namespace string) (*action.Configuration, error) {
	logger := log.FromContext(ctx)
	actionConfig := new(action.Configuration)

	// Use the in-cluster config by default
	if err := actionConfig.Init(
		c.settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			logger.V(1).Info(fmt.Sprintf(format, v...))
		},
	); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	return actionConfig, nil
}

// Install installs a Helm chart
func (c *Client) Install(ctx context.Context, releaseName, chartName, namespace string, values map[string]interface{}, version string) (*ReleaseInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger := log.FromContext(ctx).WithValues("release", releaseName, "chart", chartName, "namespace", namespace)
	logger.Info("Installing Helm chart")

	actionConfig, err := c.getActionConfig(ctx, namespace)
	if err != nil {
		return nil, err
	}

	installAction := action.NewInstall(actionConfig)
	installAction.Namespace = namespace
	installAction.ReleaseName = releaseName
	installAction.CreateNamespace = true
	installAction.Wait = false
	installAction.Timeout = 5 * time.Minute

	if version != "" {
		installAction.Version = version
	}

	chartPath, err := c.locateChart(ctx, chartName, version, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	rel, err := installAction.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %w", err)
	}

	logger.Info("Chart installed successfully", "revision", rel.Version)
	return releaseToInfo(rel), nil
}

// Upgrade upgrades an existing Helm release
func (c *Client) Upgrade(ctx context.Context, releaseName, chartName, namespace string, values map[string]interface{}, version string) (*ReleaseInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger := log.FromContext(ctx).WithValues("release", releaseName, "chart", chartName, "namespace", namespace)
	logger.Info("Upgrading Helm chart")

	actionConfig, err := c.getActionConfig(ctx, namespace)
	if err != nil {
		return nil, err
	}

	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Namespace = namespace
	upgradeAction.Wait = false
	upgradeAction.Timeout = 5 * time.Minute
	upgradeAction.ReuseValues = false

	if version != "" {
		upgradeAction.Version = version
	}

	chartPath, err := c.locateChart(ctx, chartName, version, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	rel, err := upgradeAction.RunWithContext(ctx, releaseName, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade chart: %w", err)
	}

	logger.Info("Chart upgraded successfully", "revision", rel.Version)
	return releaseToInfo(rel), nil
}

// Uninstall removes a Helm release
func (c *Client) Uninstall(ctx context.Context, releaseName, namespace string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger := log.FromContext(ctx).WithValues("release", releaseName, "namespace", namespace)
	logger.Info("Uninstalling Helm release")

	actionConfig, err := c.getActionConfig(ctx, namespace)
	if err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(actionConfig)
	uninstallAction.Timeout = 5 * time.Minute
	uninstallAction.Wait = false

	_, err = uninstallAction.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	logger.Info("Release uninstalled successfully")
	return nil
}

// GetRelease retrieves information about a Helm release
func (c *Client) GetRelease(ctx context.Context, releaseName, namespace string) (*ReleaseInfo, error) {
	actionConfig, err := c.getActionConfig(ctx, namespace)
	if err != nil {
		return nil, err
	}

	getAction := action.NewGet(actionConfig)
	rel, err := getAction.Run(releaseName)
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return releaseToInfo(rel), nil
}

// ReleaseExists checks if a release exists
func (c *Client) ReleaseExists(ctx context.Context, releaseName, namespace string) (bool, error) {
	rel, err := c.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return false, err
	}
	return rel != nil, nil
}

// locateChart finds the chart either locally or pulls it from a repository
func (c *Client) locateChart(ctx context.Context, chartName, version string, logger logr.Logger) (string, error) {
	// First, check if the chart exists locally
	localPath := filepath.Join(c.chartsPath, chartName)
	if _, err := os.Stat(localPath); err == nil {
		logger.V(1).Info("Using local chart", "path", localPath)
		return localPath, nil
	}

	// If repo URL is configured, try to pull the chart
	if c.repoURL != "" {
		return c.pullChart(ctx, chartName, version, logger)
	}

	return "", fmt.Errorf("chart %s not found locally and no repository configured", chartName)
}

// pullChart pulls a chart from the configured repository
func (c *Client) pullChart(ctx context.Context, chartName, version string, logger logr.Logger) (string, error) {
	logger.Info("Pulling chart from repository", "repo", c.repoURL)

	pullAction := action.NewPullWithOpts(action.WithConfig(new(action.Configuration)))
	pullAction.RepoURL = c.repoURL
	pullAction.Version = version
	pullAction.DestDir = c.chartsPath
	pullAction.Untar = true
	pullAction.UntarDir = c.chartsPath

	chartRef := chartName
	output, err := pullAction.Run(chartRef)
	if err != nil {
		return "", fmt.Errorf("failed to pull chart: %w", err)
	}
	logger.V(1).Info("Pull output", "output", output)

	return filepath.Join(c.chartsPath, chartName), nil
}

// AddRepository adds a Helm repository
func (c *Client) AddRepository(ctx context.Context, name, url string) error {
	logger := log.FromContext(ctx).WithValues("repo", name, "url", url)
	logger.Info("Adding Helm repository")

	repoFile := c.settings.RepositoryConfig

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(repoFile), 0755); err != nil {
		return fmt.Errorf("failed to create repo config directory: %w", err)
	}

	// Load existing repo file or create new
	f, err := repo.LoadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load repo file: %w", err)
	}
	if f == nil {
		f = repo.NewFile()
	}

	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	// Check if repo already exists
	if f.Has(name) {
		logger.Info("Repository already exists, updating")
		f.Update(entry)
	} else {
		f.Add(entry)
	}

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repo file: %w", err)
	}

	logger.Info("Repository added successfully")
	return nil
}

// GetChartMetadata returns metadata for a chart
func (c *Client) GetChartMetadata(ctx context.Context, chartName string) (*chart.Metadata, error) {
	localPath := filepath.Join(c.chartsPath, chartName)
	ch, err := loader.Load(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}
	return ch.Metadata, nil
}

// releaseToInfo converts a Helm release to ReleaseInfo
func releaseToInfo(rel *release.Release) *ReleaseInfo {
	if rel == nil {
		return nil
	}

	info := &ReleaseInfo{
		Name:      rel.Name,
		Namespace: rel.Namespace,
		Revision:  rel.Version,
		Status:    string(rel.Info.Status),
	}

	if rel.Chart != nil && rel.Chart.Metadata != nil {
		info.ChartName = rel.Chart.Metadata.Name
		info.ChartVersion = rel.Chart.Metadata.Version
		info.AppVersion = rel.Chart.Metadata.AppVersion
	}

	if rel.Info != nil && !rel.Info.LastDeployed.IsZero() {
		info.Updated = rel.Info.LastDeployed.Time
	}

	return info
}
