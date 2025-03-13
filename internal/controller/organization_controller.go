/*
Copyright 2024.

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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

const (
	oldFinalizer = "operatorkit.giantswarm.io/organization-operator-organization-controller"
	newFinalizer = "organization.giantswarm.io/finalizer"
)

var (
	organizationsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "organizations_total",
			Help: "The total number of existing organizations",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(organizationsTotal)
}

// OrganizationReconciler reconciles a Organization object
type OrganizationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Organization instance
	organization := &securityv1alpha1.Organization{}
	if err := r.Get(ctx, req.NamespacedName, organization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the Organization instance is marked to be deleted
	if organization.GetDeletionTimestamp() != nil {
		return r.reconcileDelete(ctx, organization)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(organization, newFinalizer) {
		patch := client.MergeFrom(organization.DeepCopy())
		controllerutil.AddFinalizer(organization, newFinalizer)
		if err := r.Patch(ctx, organization, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	// Create or update the Namespace
	namespaceName := fmt.Sprintf("org-%s", organization.Name)
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"giantswarm.io/organization": organization.Name,
				"giantswarm.io/managed-by":   "organization-operator",
			},
		},
	}

	if err := ctrl.SetControllerReference(organization, namespace, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to set controller reference on Namespace: %w", err)
	}

	operationResult, err := ctrl.CreateOrUpdate(ctx, r.Client, namespace, func() error {
		namespace.Labels = map[string]string{
			"giantswarm.io/organization": organization.Name,
			"giantswarm.io/managed-by":   "organization-operator",
		}
		return nil
	})

	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create or update Namespace: %w", err)
	}

	logger.Info("Namespace reconciled", "result", operationResult)

	// Update Organization status
	if organization.Status.Namespace != namespaceName {
		patch := client.MergeFrom(organization.DeepCopy())
		organization.Status.Namespace = namespaceName
		if err := r.Status().Patch(ctx, organization, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update Organization status: %w", err)
		}
	}

	if err := r.updateOrganizationCount(ctx); err != nil {
		logger.Error(err, "Failed to update organization count")
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, organization *securityv1alpha1.Organization) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Use the namespace name from the organization status
	namespaceName := organization.Status.Namespace
	if namespaceName != "" {
		// Attempt to delete the namespace without checking for its existence first
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		err := r.Delete(ctx, namespace)
		if err == nil {
			// If the namespace was found and delete was triggered, requeue
			log.Info("Namespace deletion triggered, requeuing")
			return ctrl.Result{Requeue: true}, nil
		}
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete associated namespace")
			return ctrl.Result{}, err
		}
		// If the namespace is not found, we can proceed to remove the finalizer
		log.Info("Associated namespace not found or already deleted")
	}

	// Remove old finalizer if it exists
	if controllerutil.ContainsFinalizer(organization, oldFinalizer) {
		log.Info("Removing old finalizer from Organization")
		controllerutil.RemoveFinalizer(organization, oldFinalizer)
		if err := r.Update(ctx, organization); err != nil {
			log.Error(err, "Failed to remove old finalizer")
			return ctrl.Result{}, err
		}
	}

	// Remove new finalizer if it exists
	if controllerutil.ContainsFinalizer(organization, newFinalizer) {
		log.Info("Removing new finalizer from Organization")
		controllerutil.RemoveFinalizer(organization, newFinalizer)
		if err := r.Update(ctx, organization); err != nil {
			log.Error(err, "Failed to remove new finalizer")
			return ctrl.Result{}, err
		}
	}

	if err := r.updateOrganizationCount(ctx); err != nil {
		log.Error(err, "Failed to update organization count")
		return ctrl.Result{}, err
	}

	log.Info("Organization successfully deleted")
	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) updateOrganizationCount(ctx context.Context) error {
	var organizationList securityv1alpha1.OrganizationList
	if err := r.List(ctx, &organizationList); err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}
	organizationsTotal.Set(float64(len(organizationList.Items)))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1alpha1.Organization{}).
		Owns(&corev1.Namespace{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
