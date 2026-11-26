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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

var _ = ginkgo.Describe("Organization controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	// Helper function for testing finalizer removal
	testFinalizerRemoval := func(ctx context.Context,
		name string,
		finalizer string,
		namespaceName string,
		errorMessage string) {
		org := &securityv1alpha1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Finalizers: []string{finalizer},
			},
			Spec: securityv1alpha1.OrganizationSpec{},
			Status: securityv1alpha1.OrganizationStatus{
				Namespace: namespaceName,
			},
		}
		gomega.Expect(k8sClient.Create(ctx, org)).To(gomega.Succeed())

		// Create the associated namespace
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		gomega.Expect(k8sClient.Create(ctx, namespace)).To(gomega.Succeed())

		reconciler := &OrganizationReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		// Trigger deletion
		gomega.Expect(k8sClient.Delete(ctx, org)).To(gomega.Succeed())

		// Wait for the organization to be fully deleted
		gomega.Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &securityv1alpha1.Organization{})
			if errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			// Trigger reconciliation if the organization still exists
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: name},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return fmt.Errorf("%s", errorMessage)
		}, timeout, interval).Should(gomega.Succeed())

		// Verify that the namespace has been deleted
		gomega.Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: namespaceName}, &corev1.Namespace{})
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("namespace still exists")
		}, timeout, interval).Should(gomega.Succeed())

		// Verify that the organization count metric has been updated
		initialCount := testutil.ToFloat64(organizationsTotal)
		gomega.Consistently(func() bool {
			currentCount := testutil.ToFloat64(organizationsTotal)
			return currentCount <= initialCount
		}, timeout, interval).Should(gomega.BeTrue())
	}

	ginkgo.Context("When creating and deleting Organizations", func() {
		ginkgo.It("Should create a corresponding Namespace, update the Organization status, "+
			"and update the total organizations metric", func() {
			ctx := context.Background()

			ginkgo.By("Creating the first organization")
			org1 := &securityv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-1",
				},
				Spec: securityv1alpha1.OrganizationSpec{},
			}
			gomega.Expect(k8sClient.Create(ctx, org1)).To(gomega.Succeed())

			reconciler := &OrganizationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-1"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Checking if the Namespace was created")
			namespaceName := "org-test-1"
			createdNamespace := &corev1.Namespace{}
			gomega.Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, createdNamespace)
			}, timeout, interval).Should(gomega.Succeed())
			gomega.Expect(createdNamespace.Labels).To(
				gomega.HaveKeyWithValue("giantswarm.io/organization", "test-1"),
			)
			gomega.Expect(createdNamespace.Labels).To(
				gomega.HaveKeyWithValue("giantswarm.io/managed-by", "organization-operator"),
			)

			ginkgo.By("Verifying the Organization status was updated")
			updatedOrg := &securityv1alpha1.Organization{}
			gomega.Eventually(func() string {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "test-1"}, updatedOrg)
				if err != nil {
					return ""
				}
				return updatedOrg.Status.Namespace
			}, timeout, interval).Should(gomega.Equal(namespaceName))

			ginkgo.By("Verifying the total organizations metric is 1")
			gomega.Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(gomega.Equal(float64(1)))

			ginkgo.By("Creating a second organization")
			org2 := &securityv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-2",
				},
				Spec: securityv1alpha1.OrganizationSpec{},
			}
			gomega.Expect(k8sClient.Create(ctx, org2)).To(gomega.Succeed())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-2"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Verifying the total organizations metric is 2")
			gomega.Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(gomega.Equal(float64(2)))

			ginkgo.By("Deleting the first organization")
			gomega.Expect(k8sClient.Delete(ctx, org1)).To(gomega.Succeed())

			gomega.Eventually(func() error {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "test-1"}, &securityv1alpha1.Organization{})
				if errors.IsNotFound(err) {
					return nil
				}
				if err != nil {
					return err
				}
				_, err = reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "test-1"},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				return fmt.Errorf("organization still exists")
			}, timeout, interval).Should(gomega.Succeed())

			ginkgo.By("Verifying the total organizations metric is back to 1")
			gomega.Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(gomega.Equal(float64(1)))
		})

		ginkgo.It("Should remove the finalizer when deleting an Organization", func() {
			ctx := context.Background()
			testFinalizerRemoval(
				ctx,
				"test-finalizer",
				"organization.giantswarm.io/finalizer",
				"org-test-finalizer",
				"organization still exists",
			)
		})
	})

	ginkgo.Context("When another controller writes the Organization's status", func() {
		ginkgo.It("Should keep the other writer's status fields and the Agent Platform spec", func() {
			ctx := context.Background()
			const name = "shared"

			agentPlatformSpec := &securityv1alpha1.AgentPlatformSpec{
				Enabled:   true,
				Harnesses: []string{"claude-code"},
			}
			agentPlatformStatus := &securityv1alpha1.AgentPlatformStatus{
				Namespace:      "agents-shared",
				SnapshotPrefix: "s3://snapshots/shared",
			}
			conditions := []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "Reconciled",
				LastTransitionTime: metav1.Now().Rfc3339Copy(),
			}}
			org := &securityv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       securityv1alpha1.OrganizationSpec{AgentPlatform: agentPlatformSpec},
				Status: securityv1alpha1.OrganizationStatus{
					AgentPlatform: agentPlatformStatus,
					Conditions:    conditions,
				},
			}
			// A client of its own, so the organization count of the other
			// specs does not see this organization.
			c := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithStatusSubresource(&securityv1alpha1.Organization{}).
				WithObjects(org).
				Build()
			reconciler := &OrganizationReconciler{Client: c, Scheme: c.Scheme()}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: name},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			got := &securityv1alpha1.Organization{}
			gomega.Expect(c.Get(ctx, client.ObjectKey{Name: name}, got)).To(gomega.Succeed())
			gomega.Expect(got.Status.Namespace).To(gomega.Equal("org-shared"))
			gomega.Expect(got.Status.AgentPlatform).To(gomega.Equal(agentPlatformStatus))
			// BeComparableTo compares times with time.Time.Equal: the round
			// trip through JSON gives the time the Local location, which
			// reflect.DeepEqual sees as a difference when TZ is UTC.
			gomega.Expect(got.Status.Conditions).To(gomega.BeComparableTo(conditions))
			gomega.Expect(got.Spec.AgentPlatform).To(gomega.Equal(agentPlatformSpec))
		})
	})

	ginkgo.Context("When another controller adds its finalizer after the reconciler read the Organization", func() {
		ginkgo.It("Should keep the other controller's finalizer", func() {
			ctx := context.Background()
			const (
				name           = "racing"
				otherFinalizer = "agent-platform.giantswarm.io/erasure"
			)

			otherWrote := false
			c := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithStatusSubresource(&securityv1alpha1.Organization{}).
				WithObjects(&securityv1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: name}}).
				WithInterceptorFuncs(interceptor.Funcs{
					// After the reconciler's first read, the other controller
					// adds its finalizer: the reconciler's copy is stale.
					Get: func(ctx context.Context, inner client.WithWatch, key client.ObjectKey,
						obj client.Object, opts ...client.GetOption) error {
						if err := inner.Get(ctx, key, obj, opts...); err != nil {
							return err
						}
						if _, ok := obj.(*securityv1alpha1.Organization); !ok || otherWrote {
							return nil
						}
						otherWrote = true
						current := &securityv1alpha1.Organization{}
						if err := inner.Get(ctx, key, current); err != nil {
							return err
						}
						controllerutil.AddFinalizer(current, otherFinalizer)
						return inner.Update(ctx, current)
					},
				}).
				Build()
			reconciler := &OrganizationReconciler{Client: c, Scheme: c.Scheme()}
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}

			_, err := reconciler.Reconcile(ctx, req)
			gomega.Expect(errors.IsConflict(err)).To(gomega.BeTrue(), "got error %v", err)

			got := &securityv1alpha1.Organization{}
			gomega.Expect(c.Get(ctx, client.ObjectKey{Name: name}, got)).To(gomega.Succeed())
			gomega.Expect(got.Finalizers).To(gomega.ConsistOf(otherFinalizer))

			// controller-runtime requeues on the error; the next reconcile
			// reads the current copy.
			_, err = reconciler.Reconcile(ctx, req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(c.Get(ctx, client.ObjectKey{Name: name}, got)).To(gomega.Succeed())
			gomega.Expect(got.Finalizers).To(gomega.ConsistOf(otherFinalizer, newFinalizer))
		})
	})

	ginkgo.Context("When handling Organizations with old finalizers", func() {
		ginkgo.It("Should remove the old finalizer when deleting an Organization", func() {
			ctx := context.Background()
			testFinalizerRemoval(
				ctx,
				"test-old-finalizer",
				"operatorkit.giantswarm.io/organization-operator-organization-controller",
				"org-test-old-finalizer",
				"organization with old finalizer still exists",
			)
		})
	})
})
