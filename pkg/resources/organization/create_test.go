package organization

import (
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

//	func TestNamespaceIsCreated(t *testing.T) {
//		// Set up a new scheme that includes our custom APIs
//		scheme := runtime.NewScheme()
//		_ = clientgoscheme.AddToScheme(scheme)
//		_ = securityv1alpha1.AddToScheme(scheme)
//
//		// Test parameters
//		organizationName := "giantswarm"
//		expectedNamespaceName := "org-giantswarm"
//
//		// Create a new organization
//		org := &securityv1alpha1.Organization{
//			ObjectMeta: metav1.ObjectMeta{
//				Name: organizationName,
//			},
//			Spec: securityv1alpha1.OrganizationSpec{},
//		}
//
//		// Create a secret for the organization
//		secret := &corev1.Secret{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      "credentiald-" + organizationName,
//				Namespace: "default",
//				Labels: map[string]string{
//					"app":                        "credentiald",
//					"giantswarm.io/organization": organizationName,
//				},
//			},
//			Data: map[string][]byte{
//				"azure.azureoperator.subscriptionid": []byte("test-subscription-id"),
//			},
//		}
//
//		// Create a fake client with the organization and secret
//		fakeClient := fake.NewClientBuilder().
//			WithScheme(scheme).
//			WithObjects(org, secret).
//			Build()
//
//		// Create a test logger
//		testLogger := testr.New(t)
//
//		// Create the Resource
//		r := &Resource{
//			k8sClient: fakeClient,
//			logger:    testLogger,
//		}
//
//		// Run EnsureCreated
//		ctx := context.Background()
//		err := r.EnsureCreated(ctx, org)
//		if err != nil {
//			t.Fatalf("EnsureCreated returned an error: %v", err)
//		}
//
//		// Check if the namespace was created
//		orgNamespace := &corev1.Namespace{}
//		err = fakeClient.Get(ctx, client.ObjectKey{Name: expectedNamespaceName}, orgNamespace)
//		if err != nil {
//			t.Fatalf("Failed to get created namespace: %v", err)
//		}
//
//		// Log the created namespace
//		t.Logf("Created namespace: %+v", orgNamespace)
//
//		// Check if the organization was updated
//		updatedOrg := &securityv1alpha1.Organization{}
//		err = fakeClient.Get(ctx, client.ObjectKey{Name: organizationName}, updatedOrg)
//		if err != nil {
//			t.Fatalf("Failed to get updated organization: %v", err)
//		}
//
//		// Log the updated organization
//		t.Logf("Updated organization: %+v", updatedOrg)
//
//		// Check if the namespace is stored in the organization status
//
// nolint: lll
//
//		assert.Equal(t, expectedNamespaceName, updatedOrg.Status.Namespace, "Created namespace should be stored in organization status")
//
//		// Check if the namespace has the correct labels
//		assert.Equal(t, organizationName, orgNamespace.Labels["giantswarm.io/organization"], "Namespace should have correct organization label")
//		assert.Equal(t, "organization-operator", orgNamespace.Labels["giantswarm.io/managed-by"], "Namespace should have correct managed-by label")
//
//		// Check if the organization has the correct subscription ID annotation
//		assert.Equal(t, "test-subscription-id", updatedOrg.Annotations["subscription"], "Organization should have correct subscription ID annotation")
//	}
func TestForbiddenOrganizationPrefix(t *testing.T) {
	// Set up a new scheme that includes our custom APIs
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = securityv1alpha1.AddToScheme(scheme)

	// Test parameters
	organizationName := "kube-test"
	expectedNamespaceName := "org-kube-test"

	// Create a new organization with a forbidden prefix
	org := &securityv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: organizationName,
		},
		Spec: securityv1alpha1.OrganizationSpec{},
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(org).Build()

	// Create the Resource
	r := &Resource{
		k8sClient: fakeClient,
		logger:    testr.New(t),
	}

	// Run EnsureCreated
	ctx := context.Background()
	err := r.EnsureCreated(ctx, org)
	assert.NoError(t, err, "EnsureCreated should not return an error even for forbidden prefix")

	// Check that the namespace was not created
	orgNamespace := &corev1.Namespace{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: expectedNamespaceName}, orgNamespace)
	assert.Error(t, err, "Namespace should not be created for organization with forbidden prefix")

	// Check that the organization status was not updated
	updatedOrg := &securityv1alpha1.Organization{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: organizationName}, updatedOrg)
	assert.NoError(t, err, "Should be able to get the organization")
	assert.Empty(t, updatedOrg.Status.Namespace, "Organization status should not be updated for forbidden prefix")
}
