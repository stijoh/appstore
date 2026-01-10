package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appstore "appstore/operator/api/v1alpha1"
)

// DeploymentHandler handles deployment messages by creating/updating/deleting AppDeployment CRs
type DeploymentHandler struct {
	client client.Client
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(c client.Client) *DeploymentHandler {
	return &DeploymentHandler{
		client: c,
	}
}

// HandleDeploymentRequest creates a new AppDeployment CR
func (h *DeploymentHandler) HandleDeploymentRequest(ctx context.Context, payload DeploymentRequestPayload) error {
	logger := log.FromContext(ctx).WithName("handler").WithValues(
		"requestId", payload.RequestID,
		"appName", payload.AppName,
		"namespace", payload.Namespace,
	)

	logger.Info("Handling deployment request")

	// Generate name if not provided
	name := payload.ReleaseName
	if name == "" {
		name = fmt.Sprintf("%s-%s", payload.AppName, payload.RequestID[:8])
	}

	// Convert values to JSON
	var values *apiextensionsv1.JSON
	if payload.Values != nil {
		valuesBytes, err := json.Marshal(payload.Values)
		if err != nil {
			return fmt.Errorf("failed to marshal values: %w", err)
		}
		values = &apiextensionsv1.JSON{Raw: valuesBytes}
	}

	// Create AppDeployment CR
	appDeployment := &appstore.AppDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: payload.Namespace,
			Labels: map[string]string{
				"appstore.bitpipe.no/team":       payload.TeamID,
				"appstore.bitpipe.no/app":        payload.AppName,
				"appstore.bitpipe.no/request-id": payload.RequestID,
			},
			Annotations: map[string]string{
				"appstore.bitpipe.no/requested-by": payload.UserID,
			},
		},
		Spec: appstore.AppDeploymentSpec{
			AppName:      payload.AppName,
			ChartVersion: payload.Version,
			TeamID:       payload.TeamID,
			RequestedBy:  payload.UserID,
			ReleaseName:  name,
			Values:       values,
		},
	}

	// Check if namespace exists, create if needed
	if err := h.ensureNamespace(ctx, payload.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create the AppDeployment
	if err := h.client.Create(ctx, appDeployment); err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("AppDeployment already exists", "name", name)
			return nil
		}
		return fmt.Errorf("failed to create AppDeployment: %w", err)
	}

	logger.Info("Created AppDeployment", "name", name)
	return nil
}

// HandleDeploymentUpdate updates an existing AppDeployment CR
func (h *DeploymentHandler) HandleDeploymentUpdate(ctx context.Context, payload DeploymentUpdatePayload) error {
	logger := log.FromContext(ctx).WithName("handler").WithValues(
		"requestId", payload.RequestID,
		"name", payload.Name,
		"namespace", payload.Namespace,
	)

	logger.Info("Handling deployment update")

	// Get existing AppDeployment
	appDeployment := &appstore.AppDeployment{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      payload.Name,
		Namespace: payload.Namespace,
	}, appDeployment); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("AppDeployment not found: %s/%s", payload.Namespace, payload.Name)
		}
		return fmt.Errorf("failed to get AppDeployment: %w", err)
	}

	// Verify team ownership
	if appDeployment.Spec.TeamID != payload.TeamID {
		return fmt.Errorf("team mismatch: expected %s, got %s", appDeployment.Spec.TeamID, payload.TeamID)
	}

	// Update fields
	if payload.Version != "" {
		appDeployment.Spec.ChartVersion = payload.Version
	}

	if payload.Values != nil {
		valuesBytes, err := json.Marshal(payload.Values)
		if err != nil {
			return fmt.Errorf("failed to marshal values: %w", err)
		}
		appDeployment.Spec.Values = &apiextensionsv1.JSON{Raw: valuesBytes}
	}

	// Update the AppDeployment
	if err := h.client.Update(ctx, appDeployment); err != nil {
		return fmt.Errorf("failed to update AppDeployment: %w", err)
	}

	logger.Info("Updated AppDeployment", "name", payload.Name)
	return nil
}

// HandleDeploymentDelete deletes an AppDeployment CR
func (h *DeploymentHandler) HandleDeploymentDelete(ctx context.Context, payload DeploymentDeletePayload) error {
	logger := log.FromContext(ctx).WithName("handler").WithValues(
		"requestId", payload.RequestID,
		"name", payload.Name,
		"namespace", payload.Namespace,
	)

	logger.Info("Handling deployment delete")

	// Get existing AppDeployment to verify ownership
	appDeployment := &appstore.AppDeployment{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      payload.Name,
		Namespace: payload.Namespace,
	}, appDeployment); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("AppDeployment already deleted", "name", payload.Name)
			return nil
		}
		return fmt.Errorf("failed to get AppDeployment: %w", err)
	}

	// Verify team ownership
	if appDeployment.Spec.TeamID != payload.TeamID {
		return fmt.Errorf("team mismatch: expected %s, got %s", appDeployment.Spec.TeamID, payload.TeamID)
	}

	// Delete the AppDeployment
	if err := h.client.Delete(ctx, appDeployment); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete AppDeployment: %w", err)
	}

	logger.Info("Deleted AppDeployment", "name", payload.Name)
	return nil
}

func (h *DeploymentHandler) ensureNamespace(ctx context.Context, namespace string) error {
	// For now, we assume namespaces are pre-created
	// In a production setup, you might want to create team namespaces automatically
	return nil
}
