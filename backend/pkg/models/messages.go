package models

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	// Deployment request messages
	MessageTypeDeploymentRequest MessageType = "deployment.request"
	MessageTypeDeploymentUpdate  MessageType = "deployment.update"
	MessageTypeDeploymentDelete  MessageType = "deployment.delete"

	// Status update messages (operator -> backend)
	MessageTypeStatusUpdate MessageType = "status.update"
)

// Message is the envelope for all RabbitMQ messages
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

// DeploymentRequestPayload contains the data for a deployment request
type DeploymentRequestPayload struct {
	RequestID   string                 `json:"requestId"`
	TeamID      string                 `json:"teamId"`
	UserID      string                 `json:"userId"`
	AppName     string                 `json:"appName"`
	Namespace   string                 `json:"namespace"`
	ReleaseName string                 `json:"releaseName,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Values      map[string]interface{} `json:"values,omitempty"`
}

// DeploymentUpdatePayload contains the data for updating an existing deployment
type DeploymentUpdatePayload struct {
	RequestID string                 `json:"requestId"`
	TeamID    string                 `json:"teamId"`
	UserID    string                 `json:"userId"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Version   string                 `json:"version,omitempty"`
	Values    map[string]interface{} `json:"values,omitempty"`
}

// DeploymentDeletePayload contains the data for deleting a deployment
type DeploymentDeletePayload struct {
	RequestID string `json:"requestId"`
	TeamID    string `json:"teamId"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// StatusUpdatePayload contains status updates from the operator
type StatusUpdatePayload struct {
	Name                 string    `json:"name"`
	Namespace            string    `json:"namespace"`
	Phase                string    `json:"phase"`
	Message              string    `json:"message,omitempty"`
	HelmReleaseName      string    `json:"helmReleaseName,omitempty"`
	HelmReleaseRevision  int       `json:"helmReleaseRevision,omitempty"`
	DeployedChartVersion string    `json:"deployedChartVersion,omitempty"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// Queue names
const (
	QueueDeploymentRequests = "appstore.deployments"
	QueueStatusUpdates      = "appstore.status"
)

// Exchange names
const (
	ExchangeAppstore = "appstore"
)

// Routing keys
const (
	RoutingKeyDeploymentRequest = "deployment.request"
	RoutingKeyDeploymentUpdate  = "deployment.update"
	RoutingKeyDeploymentDelete  = "deployment.delete"
	RoutingKeyStatusUpdate      = "status.update"
)
