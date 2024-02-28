package organization

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	legacyCredentialLister "github.com/giantswarm/credentiald/v2/service/lister"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/finalizerskeptcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/organization-operator/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	org, err := key.ToOrganization(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	legacyOrgName := key.LegacyOrganizationName(&org)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("listing legacy credentials for organization %#q", legacyOrgName))

	var legacyCredentials []legacyCredentialLister.Response
	{
		legacyCredentialRequest := legacyCredentialLister.Request{
			Organization: legacyOrgName,
		}
		legacyCredentials, err = r.legacyCredentialClient.List(ctx, legacyCredentialRequest)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("credentials for legacy organization %#q do not exist or were not found. %s", legacyOrgName, err))
		}
	}

	// If we allow deleting the organization without deleting
	// the credentials, then someone could create another
	// organization with the same name and re-use the credentials.
	if len(legacyCredentials) > 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found legacy credentials for organization %#q", legacyOrgName))

		finalizerskeptcontext.SetKept(ctx)

		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting legacy organization %#q", legacyOrgName))

	err = r.legacyOrgClient.DeleteCompany(legacyOrgName)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("legacy organization %#q does not exist or was not found. %s", legacyOrgName, err))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted legacy organization %#q", legacyOrgName))
	}

	orgNamespace := newOrganizationNamespace(org.Name)

	err = r.k8sClient.CtrlClient().Get(ctx, client.ObjectKey{Name: orgNamespace.Name}, orgNamespace)
	if err == nil {
		finalizerskeptcontext.SetKept(ctx)
		if orgNamespace.DeletionTimestamp != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for deletion of organization namespace %#q", orgNamespace.Name))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting organization namespace %#q", orgNamespace.Name))
			err = r.k8sClient.CtrlClient().Delete(context.Background(), orgNamespace)
		}
	}

	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("organization namespace %#q does not exist", orgNamespace.Name))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted organization namespace %#q", orgNamespace.Name))
	return nil
}
