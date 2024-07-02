package unittest

import (
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v7/pkg/k8scrdclient"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" // nolint:staticcheck

	clientgofake "k8s.io/client-go/kubernetes/fake"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

type fakeK8sClient struct {
	ctrlClient client.Client
	k8sClient  kubernetes.Interface
	scheme     *runtime.Scheme
}

func FakeK8sClient(initObjs ...runtime.Object) k8sclient.Interface {
	var err error

	var k8sClient k8sclient.Interface
	{
		scheme := runtime.NewScheme()
		err = v1.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = securityv1alpha1.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}

		k8sClient = &fakeK8sClient{
			ctrlClient: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(initObjs...).Build(),
			k8sClient:  clientgofake.NewSimpleClientset(),
			scheme:     scheme,
		}
	}

	return k8sClient
}

func (f *fakeK8sClient) CRDClient() k8scrdclient.Interface {
	return nil
}

func (f *fakeK8sClient) CtrlClient() client.Client {
	return f.ctrlClient
}

func (f *fakeK8sClient) DynClient() dynamic.Interface {
	return nil
}

func (f *fakeK8sClient) ExtClient() clientset.Interface {
	return nil
}

func (f *fakeK8sClient) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *fakeK8sClient) RESTClient() rest.Interface {
	return nil
}

func (f *fakeK8sClient) RESTConfig() *rest.Config {
	return nil
}

func (f *fakeK8sClient) Scheme() *runtime.Scheme {
	return f.scheme
}
