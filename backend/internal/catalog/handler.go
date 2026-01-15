package catalog

import (
	"encoding/json"
	"net/http"
)

// Handler handles catalog HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new catalog handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// List handles GET /api/v1/catalog
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Get optional category filter
	category := r.URL.Query().Get("category")

	var apps []App
	if category != "" {
		apps = h.service.GetAppsByCategory(category)
	} else {
		apps = h.service.ListApps()
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"apps": apps,
	})
}

// Get handles GET /api/v1/catalog/{appName}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	appName := r.PathValue("appName")
	if appName == "" {
		h.respondError(w, http.StatusBadRequest, "app name is required")
		return
	}

	app, err := h.service.GetApp(appName)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, app)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
