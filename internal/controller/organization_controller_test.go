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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

var _ = Describe("Organization controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating and deleting Organizations", func() {
		It("Should create a corresponding Namespace, update the Organization status, and update the total organizations metric", func() {
			ctx := context.Background()

			By("Creating the first organization")
			org1 := &securityv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-1",
				},
				Spec: securityv1alpha1.OrganizationSpec{},
			}
			Expect(k8sClient.Create(ctx, org1)).Should(Succeed())

			reconciler := &OrganizationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-1"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the Namespace was created")
			namespaceName := "org-test-1"
			createdNamespace := &corev1.Namespace{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, createdNamespace)
			}, timeout, interval).Should(Succeed())

			Expect(createdNamespace.Labels).To(HaveKeyWithValue("giantswarm.io/organization", "test-1"))
			Expect(createdNamespace.Labels).To(HaveKeyWithValue("giantswarm.io/managed-by", "organization-operator"))

			By("Verifying the Organization status was updated")
			updatedOrg := &securityv1alpha1.Organization{}
			Eventually(func() string {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "test-1"}, updatedOrg)
				if err != nil {
					return ""
				}
				return updatedOrg.Status.Namespace
			}, timeout, interval).Should(Equal(namespaceName))

			By("Verifying the total organizations metric is 1")
			Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(Equal(float64(1)))

			By("Creating a second organization")
			org2 := &securityv1alpha1.Organization{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "security.giantswarm.io/v1alpha1",
					Kind:       "Organization",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-2",
				},
				Spec: securityv1alpha1.OrganizationSpec{},
			}
			Expect(k8sClient.Create(ctx, org2)).Should(Succeed())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test-2"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the total organizations metric is 2")
			Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(Equal(float64(2)))

			By("Deleting the first organization")
			Expect(k8sClient.Delete(ctx, org1)).Should(Succeed())

			Eventually(func() error {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "test-1"}, &securityv1alpha1.Organization{})
				if errors.IsNotFound(err) {
					return nil
				}
				if err != nil {
					return err
				}
				// Trigger reconciliation if the organization still exists
				_, reconcileErr := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "test-1"},
				})
				if reconcileErr != nil {
					return reconcileErr
				}
				return fmt.Errorf("organization still exists")
			}, timeout, interval).Should(Succeed())

			By("Verifying the total organizations metric is back to 1")
			Eventually(func() float64 {
				return testutil.ToFloat64(organizationsTotal)
			}, timeout, interval).Should(Equal(float64(1)))
		})

	})
})
