package organization

import (
	"context"
	"fmt"
	"strings"

	companyclient "github.com/giantswarm/companyd-client-go"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/giantswarm/organization-operator/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	org, err := key.ToOrganization(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, prefix := range forbiddenOrganizationPrefixes {
		if strings.HasPrefix(org.Name, prefix) {
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("organization name %#q cannot start with %q", org.Name, prefix))
			return nil
		}
	}

	err = r.ensureOrganizationHasSubscriptionIdAnnotation(ctx, org)
	if err != nil {
		return microerror.Mask(err)
	}

	orgNamespace := newOrganizationNamespace(org.Name)
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating organization namespace %#q", orgNamespace.Name))

	err = r.k8sClient.CtrlClient().Create(ctx, orgNamespace)
	if apierrors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("organization namespace %#q already exists", orgNamespace.Name))
		err := r.ensureOrganizationNamespaceHasOrganizationLabels(ctx, orgNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created organization namespace %#q", orgNamespace.Name))

	patch := []byte(fmt.Sprintf(`{"status":{"namespace": "%s"}}`, orgNamespace.Name))
	err = r.k8sClient.CtrlClient().Status().Patch(ctx, &org, ctrl.RawPatch(types.MergePatchType, patch))
	if err != nil {
		return microerror.Mask(err)
	}

	legacyOrgName := key.LegacyOrganizationName(&org)
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating legacy organization %#q", legacyOrgName))

	legacyOrgFields := companyclient.CompanyFields{
		DefaultCluster: "deprecated",
	}
	err = r.legacyOrgClient.CreateCompany(legacyOrgName, legacyOrgFields)
	if companyclient.IsErrCompanyAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("legacy organization %#q already exists", legacyOrgName))
		return nil
	} else if err != nil {
		r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("could not create legacy organization %#q: %#q", legacyOrgName, err))
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created legacy organization %#q", legacyOrgName))

	return nil
}

func (r *Resource) ensureOrganizationHasSubscriptionIdAnnotation(ctx context.Context, organization v1alpha1.Organization) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring organization %q has subscriptionid annotation", organization.Name))
	// Retrieve secret related to this organization.
	secret, err := findSecret(ctx, r.k8sClient.CtrlClient(), organization.Name)
	if IsSecretNotFound(err) {
		// We don't want this error to block execution so we still return nil and just log the problem.
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("unable to find a secret for organization %s. Cannot set subscriptionid annotation", organization.Name))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// The subscription id field is missing in non azure installations so it's ok.
	if subscription, ok := secret.Data["azure.azureoperator.subscriptionid"]; ok && len(subscription) > 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting subscriptionid annotation to %q for organization %q", string(subscription), organization.Name))
		patch := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"subscription": "%s"}}}`, string(subscription)))
		err = r.k8sClient.CtrlClient().Patch(ctx, &organization, ctrl.RawPatch(types.MergePatchType, patch))
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("azure.azureoperator.subscriptionid field not found or empty in secret %q", secret.Name))
	}

	return nil
}

func (r *Resource) ensureOrganizationNamespaceHasOrganizationLabels(ctx context.Context, namespace *corev1.Namespace) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring organization namespace %#q has organization labels", namespace.Name))

	currentNamespace := &corev1.Namespace{}
	err := r.k8sClient.CtrlClient().Get(ctx, ctrl.ObjectKey{Name: namespace.Name}, currentNamespace)
	if err != nil {
		return microerror.Mask(err)
	}
	for key, value := range namespace.Labels {
		if currentNamespace.Labels[key] != value {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("namespace %#q has label %q=%q, but should have %q=%q", namespace.Name, key, currentNamespace.Labels[key], key, value))
			patch := []byte(fmt.Sprintf(`{"metadata":{"labels":{"%s": "%s"}}}`, key, value))
			err = r.k8sClient.CtrlClient().Patch(ctx, namespace, ctrl.RawPatch(types.MergePatchType, patch))
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured organization namespace %#q has organization labels", namespace.Name))

	return nil
}

func findSecret(ctx context.Context, client ctrl.Client, orgName string) (*corev1.Secret, error) {
	// Look for a secret with labels "app: credentiald" and "giantswarm.io/organization: org"
	secrets := &corev1.SecretList{}

	err := client.List(ctx, secrets, ctrl.MatchingLabels{"app": "credentiald", label.Organization: orgName})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(secrets.Items) > 0 {
		return &secrets.Items[0], nil
	}
	secret := &corev1.Secret{}

	// Organization-specific secret not found, use secret named "credential-default".
	err = client.Get(ctx, ctrl.ObjectKey{Namespace: "giantswarm", Name: "credential-default"}, secret)
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(secretNotFoundError, "Unable to find secret for organization %s", orgName)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}
