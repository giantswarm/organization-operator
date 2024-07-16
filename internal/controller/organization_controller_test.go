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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

var _ = Describe("Organization Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		var organization *securityv1alpha1.Organization

		BeforeEach(func() {
			organization = &securityv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: securityv1alpha1.OrganizationSpec{
					// Add any necessary spec fields
				},
			}
			Expect(k8sClient.Create(ctx, organization)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, organization)).To(Succeed())
			// Wait for the organization to be deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, &securityv1alpha1.Organization{})
				return errors.IsNotFound(err)
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})

		It("should successfully reconcile the resource", func() {
			controllerReconciler := &OrganizationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling the created resource")
			Eventually(func() error {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			By("Checking if the organization has a finalizer")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, organization)
				if err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(organization, finalizerName)
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Checking if the namespace was created")
			namespaceName := types.NamespacedName{Name: "org-" + resourceName}
			createdNamespace := &corev1.Namespace{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespaceName, createdNamespace)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())

			By("Checking if the organization status was updated")
			Eventually(func() string {
				err := k8sClient.Get(ctx, typeNamespacedName, organization)
				if err != nil {
					return ""
				}
				return organization.Status.Namespace
			}, time.Second*10, time.Millisecond*250).Should(Equal(namespaceName.Name))
		})
	})
})
