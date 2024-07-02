package organization

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/organization-operator/controllers/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	org, err := key.ToOrganization(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	orgNamespace := newOrganizationNamespace(org.Name)

	err = r.k8sClient.Get(ctx, ctrl.ObjectKey{Name: orgNamespace.Name}, orgNamespace)
	if err == nil {
		if orgNamespace.DeletionTimestamp != nil {
			r.logger.Info(fmt.Sprintf("waiting for deletion of organization namespace %#q", orgNamespace.Name))
		} else {
			r.logger.Info(fmt.Sprintf("deleting organization namespace %#q", orgNamespace.Name))
			err = r.k8sClient.Delete(ctx, orgNamespace)
		}
	}

	if apierrors.IsNotFound(err) {
		r.logger.Info(fmt.Sprintf("organization namespace %#q does not exist", orgNamespace.Name))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Info(fmt.Sprintf("deleted organization namespace %#q", orgNamespace.Name))
	return nil
}
