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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appstorev1alpha1 "appstore/operator/api/v1alpha1"
	"appstore/operator/internal/helm"
)

const (
	finalizerName = "appstore.bitpipe.no/finalizer"

	// Condition types
	ConditionTypeReady       = "Ready"
	ConditionTypeReconciling = "Reconciling"

	// Requeue intervals
	requeueAfterSuccess = 5 * time.Minute
	requeueAfterFailure = 30 * time.Second
)

// ChartValidator validates chart availability
type ChartValidator interface {
	ChartExists(chartName string) bool
	ListCharts() ([]string, error)
}

// AppDeploymentReconciler reconciles a AppDeployment object
type AppDeploymentReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	HelmClient     *helm.Client
	ChartValidator ChartValidator
}

// +kubebuilder:rbac:groups=appstore.bitpipe.no,resources=appdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appstore.bitpipe.no,resources=appdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=appstore.bitpipe.no,resources=appdeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets;configmaps;serviceaccounts;services;persistentvolumeclaims;pods;endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop for AppDeployment resources
func (r *AppDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling AppDeployment")

	// Fetch the AppDeployment instance
	appDeployment := &appstorev1alpha1.AppDeployment{}
	if err := r.Get(ctx, req.NamespacedName, appDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("AppDeployment resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get AppDeployment")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !appDeployment.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, appDeployment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(appDeployment, finalizerName) {
		controllerutil.AddFinalizer(appDeployment, finalizerName)
		if err := r.Update(ctx, appDeployment); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if suspended
	if appDeployment.Spec.Suspend {
		logger.Info("AppDeployment is suspended, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Reconcile the Helm release
	return r.reconcileHelm(ctx, appDeployment)
}

// reconcileHelm handles the Helm release installation/upgrade
func (r *AppDeploymentReconciler) reconcileHelm(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate that the requested chart exists
	if r.ChartValidator != nil && !r.ChartValidator.ChartExists(appDeployment.Spec.AppName) {
		availableCharts, _ := r.ChartValidator.ListCharts()
		msg := fmt.Sprintf("Chart '%s' not found in catalog. Available charts: %v", appDeployment.Spec.AppName, availableCharts)
		logger.Error(nil, msg)
		return r.updateStatusFailed(ctx, appDeployment, msg)
	}

	// Determine the release name
	releaseName := appDeployment.Spec.ReleaseName
	if releaseName == "" {
		releaseName = appDeployment.Name
	}

	// Get values from spec and valuesFrom
	values, err := r.getValues(ctx, appDeployment)
	if err != nil {
		return r.updateStatusFailed(ctx, appDeployment, fmt.Sprintf("Failed to get values: %v", err))
	}

	// Calculate values hash for change detection
	valuesHash := hashValues(values)

	// Check if release exists
	existingRelease, err := r.HelmClient.GetRelease(ctx, releaseName, appDeployment.Namespace)
	if err != nil {
		return r.updateStatusFailed(ctx, appDeployment, fmt.Sprintf("Failed to check existing release: %v", err))
	}

	var releaseInfo *helm.ReleaseInfo

	if existingRelease == nil {
		// Install new release
		logger.Info("Installing new Helm release", "release", releaseName, "chart", appDeployment.Spec.AppName)

		if err := r.updateStatusPhase(ctx, appDeployment, appstorev1alpha1.PhaseInstalling, "Installing Helm chart"); err != nil {
			return ctrl.Result{}, err
		}

		releaseInfo, err = r.HelmClient.Install(
			ctx,
			releaseName,
			appDeployment.Spec.AppName,
			appDeployment.Namespace,
			values,
			appDeployment.Spec.ChartVersion,
		)
		if err != nil {
			logger.Error(err, "Failed to install Helm chart")
			return r.updateStatusFailed(ctx, appDeployment, fmt.Sprintf("Failed to install: %v", err))
		}
	} else {
		// Check if upgrade is needed
		needsUpgrade := r.needsUpgrade(appDeployment, existingRelease, valuesHash)

		if needsUpgrade {
			logger.Info("Upgrading Helm release", "release", releaseName, "chart", appDeployment.Spec.AppName)

			if err := r.updateStatusPhase(ctx, appDeployment, appstorev1alpha1.PhaseUpgrading, "Upgrading Helm chart"); err != nil {
				return ctrl.Result{}, err
			}

			releaseInfo, err = r.HelmClient.Upgrade(
				ctx,
				releaseName,
				appDeployment.Spec.AppName,
				appDeployment.Namespace,
				values,
				appDeployment.Spec.ChartVersion,
			)
			if err != nil {
				logger.Error(err, "Failed to upgrade Helm chart")
				return r.updateStatusFailed(ctx, appDeployment, fmt.Sprintf("Failed to upgrade: %v", err))
			}
		} else {
			releaseInfo = existingRelease
			logger.Info("Helm release is up to date", "release", releaseName)
		}
	}

	// Update status to deployed
	return r.updateStatusDeployed(ctx, appDeployment, releaseInfo, valuesHash)
}

// reconcileDelete handles cleanup when the AppDeployment is deleted
func (r *AppDeploymentReconciler) reconcileDelete(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(appDeployment, finalizerName) {
		// Determine the release name
		releaseName := appDeployment.Spec.ReleaseName
		if releaseName == "" {
			releaseName = appDeployment.Name
		}

		// Update status to uninstalling
		if err := r.updateStatusPhase(ctx, appDeployment, appstorev1alpha1.PhaseUninstalling, "Uninstalling Helm release"); err != nil {
			return ctrl.Result{}, err
		}

		// Check if release exists before trying to uninstall
		exists, err := r.HelmClient.ReleaseExists(ctx, releaseName, appDeployment.Namespace)
		if err != nil {
			logger.Error(err, "Failed to check if release exists")
			return ctrl.Result{RequeueAfter: requeueAfterFailure}, err
		}

		if exists {
			logger.Info("Uninstalling Helm release", "release", releaseName)
			if err := r.HelmClient.Uninstall(ctx, releaseName, appDeployment.Namespace); err != nil {
				logger.Error(err, "Failed to uninstall Helm release")
				return ctrl.Result{RequeueAfter: requeueAfterFailure}, err
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(appDeployment, finalizerName)
		if err := r.Update(ctx, appDeployment); err != nil {
			logger.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getValues retrieves and merges values from spec and valuesFrom references
func (r *AppDeploymentReconciler) getValues(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	// Get values from valuesFrom references first
	for _, ref := range appDeployment.Spec.ValuesFrom {
		refValues, err := r.getValuesFromReference(ctx, appDeployment.Namespace, ref)
		if err != nil {
			if ref.Optional {
				continue
			}
			return nil, fmt.Errorf("failed to get values from %s/%s: %w", ref.Kind, ref.Name, err)
		}
		values = mergeMaps(values, refValues)
	}

	// Merge spec values (these take precedence)
	if appDeployment.Spec.Values != nil {
		var specValues map[string]interface{}
		if err := json.Unmarshal(appDeployment.Spec.Values.Raw, &specValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec values: %w", err)
		}
		values = mergeMaps(values, specValues)
	}

	return values, nil
}

// getValuesFromReference retrieves values from a ConfigMap or Secret
func (r *AppDeploymentReconciler) getValuesFromReference(ctx context.Context, namespace string, ref appstorev1alpha1.ValuesReference) (map[string]interface{}, error) {
	key := ref.ValuesKey
	if key == "" {
		key = "values.yaml"
	}

	var data string

	switch ref.Kind {
	case "ConfigMap":
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, cm); err != nil {
			return nil, err
		}
		var ok bool
		data, ok = cm.Data[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in ConfigMap %s", key, ref.Name)
		}

	case "Secret":
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, secret); err != nil {
			return nil, err
		}
		dataBytes, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in Secret %s", key, ref.Name)
		}
		data = string(dataBytes)

	default:
		return nil, fmt.Errorf("unsupported kind: %s", ref.Kind)
	}

	var values map[string]interface{}
	if err := json.Unmarshal([]byte(data), &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal values: %w", err)
	}

	return values, nil
}

// needsUpgrade determines if the Helm release needs to be upgraded
func (r *AppDeploymentReconciler) needsUpgrade(appDeployment *appstorev1alpha1.AppDeployment, release *helm.ReleaseInfo, valuesHash string) bool {
	// Check if values changed
	if appDeployment.Status.LastAppliedValuesHash != valuesHash {
		return true
	}

	// Check if chart version changed
	if appDeployment.Spec.ChartVersion != "" && appDeployment.Spec.ChartVersion != release.ChartVersion {
		return true
	}

	return false
}

// updateStatusPhase updates the status phase
func (r *AppDeploymentReconciler) updateStatusPhase(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment, phase appstorev1alpha1.AppDeploymentPhase, message string) error {
	appDeployment.Status.Phase = phase
	appDeployment.Status.Message = message
	appDeployment.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	appDeployment.Status.ObservedGeneration = appDeployment.Generation

	meta.SetStatusCondition(&appDeployment.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             string(phase),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})

	return r.Status().Update(ctx, appDeployment)
}

// updateStatusDeployed updates the status after successful deployment
func (r *AppDeploymentReconciler) updateStatusDeployed(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment, releaseInfo *helm.ReleaseInfo, valuesHash string) (ctrl.Result, error) {
	appDeployment.Status.Phase = appstorev1alpha1.PhaseDeployed
	appDeployment.Status.Message = "Helm release deployed successfully"
	appDeployment.Status.HelmReleaseName = releaseInfo.Name
	appDeployment.Status.HelmReleaseRevision = releaseInfo.Revision
	appDeployment.Status.DeployedChartVersion = releaseInfo.ChartVersion
	appDeployment.Status.LastAppliedValuesHash = valuesHash
	appDeployment.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	appDeployment.Status.ObservedGeneration = appDeployment.Generation
	appDeployment.Status.FailureCount = 0

	meta.SetStatusCondition(&appDeployment.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Deployed",
		Message:            "Helm release is deployed and ready",
		LastTransitionTime: metav1.Now(),
	})

	meta.SetStatusCondition(&appDeployment.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReconciling,
		Status:             metav1.ConditionFalse,
		Reason:             "Deployed",
		Message:            "Reconciliation complete",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, appDeployment); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfterSuccess}, nil
}

// updateStatusFailed updates the status after a failure
func (r *AppDeploymentReconciler) updateStatusFailed(ctx context.Context, appDeployment *appstorev1alpha1.AppDeployment, message string) (ctrl.Result, error) {
	appDeployment.Status.Phase = appstorev1alpha1.PhaseFailed
	appDeployment.Status.Message = message
	appDeployment.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	appDeployment.Status.ObservedGeneration = appDeployment.Generation
	appDeployment.Status.FailureCount++

	meta.SetStatusCondition(&appDeployment.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "Failed",
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, appDeployment); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil
}

// hashValues creates a SHA256 hash of the values map
func hashValues(values map[string]interface{}) string {
	data, _ := json.Marshal(values)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8])
}

// mergeMaps recursively merges src into dst
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			srcMap, srcOk := srcVal.(map[string]interface{})
			dstMap, dstOk := dstVal.(map[string]interface{})
			if srcOk && dstOk {
				dst[key] = mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[key] = srcVal
	}
	return dst
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appstorev1alpha1.AppDeployment{}).
		Named("appdeployment").
		Complete(r)
}
