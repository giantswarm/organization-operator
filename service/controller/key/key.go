package key

import (
	securityv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
)

func ToOrganization(v interface{}) (securityv1alpha1.Organization, error) {
	if v == nil {
		return securityv1alpha1.Organization{}, microerror.Maskf(wrongTypeError, "expected non-nil, got %#v", v)
	}

	p, ok := v.(*securityv1alpha1.Organization)
	if !ok {
		return securityv1alpha1.Organization{}, microerror.Maskf(wrongTypeError, "expected %T, got %T", p, v)
	}

	c := p.DeepCopy()

	return *c, nil
}

func LegacyOrganizationName(cr *securityv1alpha1.Organization) string {
	if cr.GetAnnotations() != nil {
		name := cr.GetAnnotations()[annotation.UIOriginalOrganizationName]
		if len(name) > 0 {
			return name
		}
	}

	return cr.GetName()
}
