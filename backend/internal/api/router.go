package api

import (
	"net/http"

	"appstore/backend/internal/deployment"
	"appstore/backend/internal/rabbitmq"
)

// Router sets up HTTP routes
type Router struct {
	mux               *http.ServeMux
	deploymentHandler *deployment.Handler
}

// NewRouter creates a new router with all handlers
func NewRouter(publisher *rabbitmq.Publisher) *Router {
	r := &Router{
		mux:               http.NewServeMux(),
		deploymentHandler: deployment.NewHandler(publisher),
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Health check
	r.mux.HandleFunc("GET /healthz", r.healthz)

	// API v1 routes
	r.mux.HandleFunc("POST /api/v1/deployments", r.deploymentHandler.Create)
	r.mux.HandleFunc("GET /api/v1/deployments", r.deploymentHandler.List)
	r.mux.HandleFunc("GET /api/v1/deployments/{id}", r.deploymentHandler.Get)
	r.mux.HandleFunc("PUT /api/v1/deployments/{id}", r.deploymentHandler.Update)
	r.mux.HandleFunc("DELETE /api/v1/deployments/{id}", r.deploymentHandler.Delete)
}

func (r *Router) healthz(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
