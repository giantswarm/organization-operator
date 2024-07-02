package key

import (
	"github.com/giantswarm/microerror"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
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
