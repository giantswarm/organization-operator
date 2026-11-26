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
	"os"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/controller/openapi/builder"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	clientgoapplyconfigurations "k8s.io/client-go/applyconfigurations"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

const crdPath = "../../config/crd/bases/security.giantswarm.io_organizations.yaml"

var (
	k8sClient client.Client

	// typeConverters let the fake client's server-side apply merge the
	// lists of an Organization by the list types of the generated CRD, as
	// the API server does. Its default converter takes every list of a
	// custom resource as atomic.
	typeConverters []managedfields.TypeConverter
)

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controller Suite")
}

// newClientBuilder starts a fake client that serves the Organization's status
// subresource and applies by the generated CRD's schema.
func newClientBuilder() *fake.ClientBuilder {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithStatusSubresource(&securityv1alpha1.Organization{}).
		WithTypeConverters(typeConverters...)
}

// organizationTypeConverter builds the type converter of the generated CRD
// the way the API server builds it for a custom resource.
func organizationTypeConverter() (managedfields.TypeConverter, error) {
	raw, err := os.ReadFile(crdPath)
	if err != nil {
		return nil, err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.UnmarshalStrict(raw, crd); err != nil {
		return nil, err
	}
	openAPI, err := builder.BuildOpenAPIV3(crd, crd.Spec.Versions[0].Name, builder.Options{})
	if err != nil {
		return nil, err
	}
	return managedfields.NewTypeConverter(openAPI.Components.Schemas, false)
}

var _ = ginkgo.BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	ginkgo.By("bootstrapping test environment")

	err := securityv1alpha1.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// The client-go converter knows the built-in kinds, such as Namespace,
	// and refuses the others; a scheme of the built-in kinds alone keeps it
	// from claiming the Organization.
	builtInScheme := runtime.NewScheme()
	gomega.Expect(scheme.AddToScheme(builtInScheme)).To(gomega.Succeed())
	organizationConverter, err := organizationTypeConverter()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	typeConverters = []managedfields.TypeConverter{
		clientgoapplyconfigurations.NewTypeConverter(builtInScheme),
		organizationConverter,
	}

	k8sClient = newClientBuilder().Build()
	gomega.Expect(k8sClient).NotTo(gomega.BeNil())
})
