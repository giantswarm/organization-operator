package organization

import (
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/organization-operator/api/v1alpha1"
)

func TestResource_EnsureDeleted(t *testing.T) {
	testCases := []struct {
		name             string
		organizationName string
		namespaceState   string
		expectError      bool
	}{
		{
			name:             "case 0: Namespace exists and is not being deleted",
			organizationName: "test-org",
			namespaceState:   "exists",
			expectError:      false,
		},
		{
			name:             "case 1: Namespace exists and is being deleted",
			organizationName: "test-org",
			namespaceState:   "deleting",
			expectError:      false,
		},
		{
			name:             "case 2: Namespace does not exist",
			organizationName: "test-org",
			namespaceState:   "not-exists",
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(scheme)
			require.NoError(t, err)
			err = v1alpha1.AddToScheme(scheme)
			require.NoError(t, err)

			org := newOrg(tc.organizationName)
			namespace := newOrganizationNamespace(tc.organizationName)

			var objs []runtime.Object
			objs = append(objs, org)

			if tc.namespaceState != "not-exists" {
				if tc.namespaceState == "deleting" {
					now := metav1.Now()
					namespace.DeletionTimestamp = &now
					// Add a finalizer to make it a valid object
					namespace.Finalizers = []string{"test-finalizer"}
				}
				objs = append(objs, namespace)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
			r := &Resource{
				k8sClient: fakeClient,
				logger:    testr.New(t),
			}

			err = r.EnsureDeleted(context.Background(), org)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check the state of the namespace after EnsureDeleted
			var resultNamespace corev1.Namespace
			err = fakeClient.Get(context.Background(), client.ObjectKey{Name: namespace.Name}, &resultNamespace)

			switch tc.namespaceState {
			case "exists":
				assert.True(t, errors.IsNotFound(err), "Namespace should be deleted")
			case "deleting":
				assert.NoError(t, err, "Namespace should still exist")
				assert.NotNil(t, resultNamespace.DeletionTimestamp, "Namespace should still be marked for deletion")
				assert.NotEmpty(t, resultNamespace.Finalizers, "Namespace should still have finalizers")
			case "not-exists":
				assert.True(t, errors.IsNotFound(err), "Namespace should not exist")
			}
		})
	}
}
