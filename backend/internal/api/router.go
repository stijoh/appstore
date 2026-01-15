package api

import (
	"net/http"

	"appstore/backend/internal/catalog"
	"appstore/backend/internal/deployment"
	"appstore/backend/internal/k8s"
	"appstore/backend/internal/rabbitmq"
)

// Router sets up HTTP routes
type Router struct {
	mux               *http.ServeMux
	deploymentHandler *deployment.Handler
	catalogHandler    *catalog.Handler
}

// NewRouter creates a new router with all handlers
func NewRouter(publisher *rabbitmq.Publisher, k8sClient *k8s.Client, catalogService *catalog.Service) *Router {
	r := &Router{
		mux:               http.NewServeMux(),
		deploymentHandler: deployment.NewHandler(publisher, k8sClient),
		catalogHandler:    catalog.NewHandler(catalogService),
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Health check
	r.mux.HandleFunc("GET /healthz", r.healthz)

	// Catalog routes
	r.mux.HandleFunc("GET /api/v1/catalog", r.catalogHandler.List)
	r.mux.HandleFunc("GET /api/v1/catalog/{appName}", r.catalogHandler.Get)

	// Deployment routes
	r.mux.HandleFunc("POST /api/v1/deployments", r.deploymentHandler.Create)
	r.mux.HandleFunc("GET /api/v1/deployments", r.deploymentHandler.List)
	r.mux.HandleFunc("GET /api/v1/deployments/{name}", r.deploymentHandler.Get)
	r.mux.HandleFunc("PUT /api/v1/deployments/{name}", r.deploymentHandler.Update)
	r.mux.HandleFunc("DELETE /api/v1/deployments/{name}", r.deploymentHandler.Delete)
}

func (r *Router) healthz(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ServeHTTP implements http.Handler with CORS support
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	r.mux.ServeHTTP(w, req)
}
