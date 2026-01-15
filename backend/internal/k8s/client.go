package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// AppDeploymentGVR is the GroupVersionResource for AppDeployment
var AppDeploymentGVR = schema.GroupVersionResource{
	Group:    "appstore.bitpipe.no",
	Version:  "v1alpha1",
	Resource: "appdeployments",
}

// Condition represents a Kubernetes condition
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// AppDeployment represents an AppDeployment resource
type AppDeployment struct {
	Name                 string      `json:"name"`
	Namespace            string      `json:"namespace"`
	AppName              string      `json:"appName"`
	ChartVersion         string      `json:"chartVersion,omitempty"`
	TeamID               string      `json:"teamId"`
	RequestedBy          string      `json:"requestedBy,omitempty"`
	Phase                string      `json:"phase"`
	HelmReleaseName      string      `json:"helmReleaseName,omitempty"`
	HelmReleaseRevision  int64       `json:"helmReleaseRevision,omitempty"`
	DeployedChartVersion string      `json:"deployedChartVersion,omitempty"`
	Message              string      `json:"message,omitempty"`
	Conditions           []Condition `json:"conditions,omitempty"`
	CreatedAt            time.Time   `json:"createdAt"`
	LastReconcileTime    *time.Time  `json:"lastReconcileTime,omitempty"`
}

// Client provides access to Kubernetes resources
type Client struct {
	dynamicClient dynamic.Interface
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
			config, err = kubeConfig.ClientConfig()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
	}, nil
}

// ListAppDeployments returns all AppDeployments in a namespace (or all namespaces if empty)
func (c *Client) ListAppDeployments(ctx context.Context, namespace string) ([]AppDeployment, error) {
	var list *unstructured.UnstructuredList
	var err error

	if namespace != "" {
		list, err = c.dynamicClient.Resource(AppDeploymentGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamicClient.Resource(AppDeploymentGVR).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list AppDeployments: %w", err)
	}

	var deployments []AppDeployment
	for _, item := range list.Items {
		deployment, err := parseAppDeployment(&item)
		if err != nil {
			continue // Skip items that can't be parsed
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

// GetAppDeployment returns a specific AppDeployment
func (c *Client) GetAppDeployment(ctx context.Context, namespace, name string) (*AppDeployment, error) {
	item, err := c.dynamicClient.Resource(AppDeploymentGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get AppDeployment: %w", err)
	}

	return parseAppDeployment(item)
}

func parseAppDeployment(item *unstructured.Unstructured) (*AppDeployment, error) {
	deployment := &AppDeployment{
		Name:      item.GetName(),
		Namespace: item.GetNamespace(),
		CreatedAt: item.GetCreationTimestamp().Time,
	}

	// Parse spec
	spec, found, err := unstructured.NestedMap(item.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get spec")
	}

	if appName, ok := spec["appName"].(string); ok {
		deployment.AppName = appName
	}
	if chartVersion, ok := spec["chartVersion"].(string); ok {
		deployment.ChartVersion = chartVersion
	}
	if teamID, ok := spec["teamId"].(string); ok {
		deployment.TeamID = teamID
	}
	if requestedBy, ok := spec["requestedBy"].(string); ok {
		deployment.RequestedBy = requestedBy
	}

	// Parse status
	status, found, _ := unstructured.NestedMap(item.Object, "status")
	if found {
		if phase, ok := status["phase"].(string); ok {
			deployment.Phase = phase
		}
		if helmReleaseName, ok := status["helmReleaseName"].(string); ok {
			deployment.HelmReleaseName = helmReleaseName
		}
		if helmReleaseRevision, ok := status["helmReleaseRevision"].(int64); ok {
			deployment.HelmReleaseRevision = helmReleaseRevision
		}
		if deployedChartVersion, ok := status["deployedChartVersion"].(string); ok {
			deployment.DeployedChartVersion = deployedChartVersion
		}
		if message, ok := status["message"].(string); ok {
			deployment.Message = message
		}

		// Parse lastReconcileTime
		if lastReconcileTime, ok := status["lastReconcileTime"].(string); ok {
			if t, err := time.Parse(time.RFC3339, lastReconcileTime); err == nil {
				deployment.LastReconcileTime = &t
			}
		}

		// Parse conditions
		if conditions, ok := status["conditions"].([]interface{}); ok {
			for _, c := range conditions {
				if condMap, ok := c.(map[string]interface{}); ok {
					cond := Condition{}
					if t, ok := condMap["type"].(string); ok {
						cond.Type = t
					}
					if s, ok := condMap["status"].(string); ok {
						cond.Status = s
					}
					if r, ok := condMap["reason"].(string); ok {
						cond.Reason = r
					}
					if m, ok := condMap["message"].(string); ok {
						cond.Message = m
					}
					if ltt, ok := condMap["lastTransitionTime"].(string); ok {
						if t, err := time.Parse(time.RFC3339, ltt); err == nil {
							cond.LastTransitionTime = t
						}
					}
					deployment.Conditions = append(deployment.Conditions, cond)
				}
			}
		}
	}

	return deployment, nil
}
