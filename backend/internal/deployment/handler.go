package deployment

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"appstore/backend/internal/k8s"
	"appstore/backend/internal/rabbitmq"
	"appstore/backend/pkg/models"
)

// CreateRequest is the request body for creating a deployment
type CreateRequest struct {
	AppName     string                 `json:"appName"`
	Namespace   string                 `json:"namespace"`
	ReleaseName string                 `json:"releaseName,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Values      map[string]interface{} `json:"values,omitempty"`
}

// UpdateRequest is the request body for updating a deployment
type UpdateRequest struct {
	Version string                 `json:"version,omitempty"`
	Values  map[string]interface{} `json:"values,omitempty"`
}

// Handler handles deployment HTTP requests
type Handler struct {
	publisher *rabbitmq.Publisher
	k8sClient *k8s.Client
	logger    *slog.Logger
}

// NewHandler creates a new deployment handler
func NewHandler(publisher *rabbitmq.Publisher, k8sClient *k8s.Client) *Handler {
	return &Handler{
		publisher: publisher,
		k8sClient: k8sClient,
		logger:    slog.Default().With("component", "deployment-handler"),
	}
}

// Create handles POST /api/v1/deployments
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if h.publisher == nil {
		h.respondError(w, http.StatusServiceUnavailable, "RabbitMQ not available")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.AppName == "" {
		h.respondError(w, http.StatusBadRequest, "appName is required")
		return
	}
	if req.Namespace == "" {
		h.respondError(w, http.StatusBadRequest, "namespace is required")
		return
	}

	// TODO: Get team ID and user ID from auth context
	teamID := "default-team"
	userID := "anonymous"

	requestID := uuid.New().String()

	payload := models.DeploymentRequestPayload{
		RequestID:   requestID,
		TeamID:      teamID,
		UserID:      userID,
		AppName:     req.AppName,
		Namespace:   req.Namespace,
		ReleaseName: req.ReleaseName,
		Version:     req.Version,
		Values:      req.Values,
	}

	if err := h.publisher.PublishDeploymentRequest(r.Context(), payload); err != nil {
		h.logger.Error("failed to publish deployment request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	h.logger.Info("deployment request published",
		"requestId", requestID,
		"appName", req.AppName,
		"namespace", req.Namespace,
	)

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"requestId": requestID,
		"message":   "deployment request accepted",
	})
}

// List handles GET /api/v1/deployments
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Kubernetes not available")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	deployments, err := h.k8sClient.ListAppDeployments(r.Context(), namespace)
	if err != nil {
		h.logger.Error("failed to list deployments", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"deployments": deployments,
	})
}

// Get handles GET /api/v1/deployments/{name}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Kubernetes not available")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "deployment name is required")
		return
	}

	// Default to "default" namespace, can be overridden with query param
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	deployment, err := h.k8sClient.GetAppDeployment(r.Context(), namespace, name)
	if err != nil {
		h.logger.Error("failed to get deployment", "error", err, "name", name, "namespace", namespace)
		h.respondError(w, http.StatusNotFound, "deployment not found")
		return
	}

	h.respondJSON(w, http.StatusOK, deployment)
}

// Update handles PUT /api/v1/deployments/{name}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil || h.publisher == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Kubernetes or RabbitMQ not available")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "deployment name is required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default to "default" namespace, can be overridden with query param
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Verify deployment exists and get its details
	deployment, err := h.k8sClient.GetAppDeployment(r.Context(), namespace, name)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "deployment not found")
		return
	}

	// TODO: Get team ID and user ID from auth context
	teamID := deployment.TeamID
	userID := "anonymous"

	requestID := uuid.New().String()

	payload := models.DeploymentUpdatePayload{
		RequestID: requestID,
		TeamID:    teamID,
		UserID:    userID,
		Name:      name,
		Namespace: namespace,
		Version:   req.Version,
		Values:    req.Values,
	}

	if err := h.publisher.PublishDeploymentUpdate(r.Context(), payload); err != nil {
		h.logger.Error("failed to publish deployment update", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to update deployment")
		return
	}

	h.logger.Info("deployment update published",
		"requestId", requestID,
		"name", name,
	)

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"requestId": requestID,
		"message":   "deployment update request accepted",
	})
}

// Delete handles DELETE /api/v1/deployments/{name}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil || h.publisher == nil {
		h.respondError(w, http.StatusServiceUnavailable, "Kubernetes or RabbitMQ not available")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "deployment name is required")
		return
	}

	// Default to "default" namespace, can be overridden with query param
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Verify deployment exists and get its details
	deployment, err := h.k8sClient.GetAppDeployment(r.Context(), namespace, name)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "deployment not found")
		return
	}

	// TODO: Get team ID and user ID from auth context
	teamID := deployment.TeamID
	userID := "anonymous"

	requestID := uuid.New().String()

	payload := models.DeploymentDeletePayload{
		RequestID: requestID,
		TeamID:    teamID,
		UserID:    userID,
		Name:      name,
		Namespace: namespace,
	}

	if err := h.publisher.PublishDeploymentDelete(r.Context(), payload); err != nil {
		h.logger.Error("failed to publish deployment delete", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to delete deployment")
		return
	}

	h.logger.Info("deployment delete published",
		"requestId", requestID,
		"name", name,
	)

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"requestId": requestID,
		"message":   "deployment delete request accepted",
	})
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
