package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

const (
	testFinalizer = "test.finalizers.giantswarm.io/test-finalizer"
	orgFinalizer  = "security.giantswarm.io/organization-finalizer"
)

func TestOrganizationReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name                string
		organizationName    string
		namespaceFinalizers []string
		expectNsExists      bool
		expectNsLabels      map[string]string
	}{
		{
			name:                "case 0: Create namespace when there is no org namespace",
			organizationName:    "giantswarm",
			namespaceFinalizers: nil,
			expectNsExists:      true,
			expectNsLabels: map[string]string{
				"giantswarm.io/organization": "giantswarm",
				"giantswarm.io/managed-by":   "organization-operator",
			},
		},
		{
			name:                "case 1: Update namespace when it has no finalizers",
			organizationName:    "giantswarm",
			namespaceFinalizers: []string{},
			expectNsExists:      true,
			expectNsLabels: map[string]string{
				"giantswarm.io/organization": "giantswarm",
				"giantswarm.io/managed-by":   "organization-operator",
			},
		},
		{
			name:                "case 2: Update namespace when it has finalizers",
			organizationName:    "giantswarm",
			namespaceFinalizers: []string{testFinalizer},
			expectNsExists:      true,
			expectNsLabels: map[string]string{
				"giantswarm.io/organization": "giantswarm",
				"giantswarm.io/managed-by":   "organization-operator",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace := newOrganizationNamespace(tc.organizationName, tc.namespaceFinalizers)
			org := newOrg(tc.organizationName, namespace.Name)

			scheme := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(scheme)
			require.NoError(t, err)
			err = securityv1alpha1.AddToScheme(scheme)
			require.NoError(t, err)

			objs := []runtime.Object{org}
			if tc.namespaceFinalizers != nil {
				objs = append(objs, namespace)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
			reconciler := &OrganizationReconciler{
				Client: fakeClient,
				Log:    testr.New(t),
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: tc.organizationName,
				},
			}

			_, err = reconciler.Reconcile(context.Background(), req)
			// We expect an error because the organization is not found after creation
			assert.Error(t, err, "Reconcile should return an error")
			// nolint: lll
			assert.Contains(t, err.Error(), "organizations.security.giantswarm.io \"giantswarm\" not found", "Error should indicate organization not found")

			// Check namespace state
			var resultNamespace corev1.Namespace
			err = fakeClient.Get(context.Background(), client.ObjectKey{Name: namespace.Name}, &resultNamespace)
			if tc.expectNsExists {
				assert.NoError(t, err, "Expected to find the namespace")
				assert.Equal(t, tc.expectNsLabels, resultNamespace.Labels, "Namespace labels should match expected labels")
				if tc.namespaceFinalizers != nil {
					assert.ElementsMatch(t, tc.namespaceFinalizers, resultNamespace.Finalizers, "Namespace finalizers should match")
				}
			} else {
				assert.True(t, errors.IsNotFound(err), "Expected namespace to not exist")
			}

			// Log the final state for debugging
			t.Logf("Final namespace state: %+v", resultNamespace)
		})
	}
}

func TestOrganizationReconciler_ReconcileDelete(t *testing.T) {
	testCases := []struct {
		name                string
		organizationName    string
		namespaceFinalizers []string
		expectedIterations  int
	}{
		{
			name:                "case 0: Delete organization after deleting namespace",
			organizationName:    "giantswarm",
			namespaceFinalizers: []string{testFinalizer},
			expectedIterations:  3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace := newOrganizationNamespace(tc.organizationName, tc.namespaceFinalizers)
			org := newOrgForDeletion(tc.organizationName, namespace.Name)

			scheme := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(scheme)
			require.NoError(t, err)
			err = securityv1alpha1.AddToScheme(scheme)
			require.NoError(t, err)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(org, namespace).Build()
			reconciler := &OrganizationReconciler{
				Client: fakeClient,
				Log:    testr.New(t),
				Scheme: scheme,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: tc.organizationName,
				},
			}

			for i := 0; i < tc.expectedIterations; i++ {
				_, err := reconciler.Reconcile(context.Background(), req)
				require.NoError(t, err, "Reconcile should not return an error")

				if i == 0 {
					// After first iteration, remove namespace finalizers
					var ns corev1.Namespace
					err = fakeClient.Get(context.Background(), client.ObjectKey{Name: namespace.Name}, &ns)
					if err == nil {
						ns.Finalizers = []string{}
						err = fakeClient.Update(context.Background(), &ns)
						require.NoError(t, err, "Failed to update namespace")
					}
				}

				// Check if both organization and namespace are deleted
				// nolint: lll
				orgErr := fakeClient.Get(context.Background(), client.ObjectKey{Name: tc.organizationName}, &securityv1alpha1.Organization{})
				nsErr := fakeClient.Get(context.Background(), client.ObjectKey{Name: namespace.Name}, &corev1.Namespace{})

				if errors.IsNotFound(orgErr) && errors.IsNotFound(nsErr) {
					return // Test passed, both org and namespace are deleted
				}
			}

			t.Errorf("Failed to delete organization and namespace within %d iterations", tc.expectedIterations)
		})
	}
}

func newOrg(name, namespace string) *securityv1alpha1.Organization {
	return &securityv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Finalizers: []string{orgFinalizer},
		},
		Spec: securityv1alpha1.OrganizationSpec{},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: namespace,
		},
	}
}

func newOrgForDeletion(name, namespace string) *securityv1alpha1.Organization {
	return &securityv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Finalizers:        []string{orgFinalizer},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: securityv1alpha1.OrganizationSpec{},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: namespace,
		},
	}
}

func newOrganizationNamespace(orgName string, finalizers []string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:       fmt.Sprintf("org-%s", orgName),
			Finalizers: finalizers,
		},
	}
}
