package catalog

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// App represents an application in the catalog
type App struct {
	Name        string   `json:"name" yaml:"name"`
	DisplayName string   `json:"displayName" yaml:"displayName"`
	Description string   `json:"description" yaml:"description"`
	Icon        string   `json:"icon" yaml:"icon"`
	Category    string   `json:"category" yaml:"category"`
	ChartPath   string   `json:"chartPath" yaml:"chartPath"`
	Tags        []string `json:"tags" yaml:"tags"`
}

// Catalog represents the full catalog of available apps
type Catalog struct {
	Apps []App `json:"apps" yaml:"apps"`
}

// Service provides access to the app catalog
type Service struct {
	catalogPath string
	catalog     *Catalog
	mu          sync.RWMutex
}

// NewService creates a new catalog service
func NewService(catalogPath string) *Service {
	return &Service{
		catalogPath: catalogPath,
	}
}

// Load reads and parses the catalog file
func (s *Service) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog file: %w", err)
	}

	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse catalog file: %w", err)
	}

	s.catalog = &catalog
	return nil
}

// ListApps returns all apps in the catalog
func (s *Service) ListApps() []App {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.catalog == nil {
		return []App{}
	}

	return s.catalog.Apps
}

// GetApp returns a specific app by name
func (s *Service) GetApp(name string) (*App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.catalog == nil {
		return nil, fmt.Errorf("catalog not loaded")
	}

	for _, app := range s.catalog.Apps {
		if app.Name == name {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("app not found: %s", name)
}

// GetAppsByCategory returns all apps in a specific category
func (s *Service) GetAppsByCategory(category string) []App {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.catalog == nil {
		return []App{}
	}

	var apps []App
	for _, app := range s.catalog.Apps {
		if app.Category == category {
			apps = append(apps, app)
		}
	}

	return apps
}

// AppExists checks if an app exists in the catalog
func (s *Service) AppExists(name string) bool {
	_, err := s.GetApp(name)
	return err == nil
}
