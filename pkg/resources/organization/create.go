package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/organization-operator/controllers/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	org, err := key.ToOrganization(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, prefix := range forbiddenOrganizationPrefixes {
		if strings.HasPrefix(org.Name, prefix) {
			r.logger.Info(fmt.Sprintf("organization name %#q cannot start with %q", org.Name, prefix))
			return nil
		}
	}

	orgNamespace := newOrganizationNamespace(org.Name)
	r.logger.Info(fmt.Sprintf("creating organization namespace %#q", orgNamespace.Name))

	err = r.k8sClient.Create(ctx, orgNamespace)
	if apierrors.IsAlreadyExists(err) {
		r.logger.Info(fmt.Sprintf("organization namespace %#q already exists", orgNamespace.Name))
		err := r.ensureOrganizationNamespaceHasOrganizationLabels(ctx, orgNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Info(fmt.Sprintf("created organization namespace %#q", orgNamespace.Name))

	patch := []byte(fmt.Sprintf(`{"status":{"namespace": "%s"}}`, orgNamespace.Name))
	err = r.k8sClient.Status().Patch(ctx, &org, ctrl.RawPatch(types.MergePatchType, patch))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureOrganizationNamespaceHasOrganizationLabels(ctx context.Context,
	namespace *corev1.Namespace) error {
	r.logger.Info(fmt.Sprintf("ensuring organization namespace %#q has organization labels", namespace.Name))

	currentNamespace := &corev1.Namespace{}
	err := r.k8sClient.Get(ctx, ctrl.ObjectKey{Name: namespace.Name}, currentNamespace)
	if err != nil {
		return microerror.Mask(err)
	}
	for key, value := range namespace.Labels {
		if currentNamespace.Labels[key] != value {
			r.logger.Info(fmt.Sprintf("namespace %#q has label %q=%q, but should have %q=%q",
				namespace.Name, key, currentNamespace.Labels[key], key, value))
			patch := []byte(fmt.Sprintf(`{"metadata":{"labels":{"%s": "%s"}}}`, key, value))
			err = r.k8sClient.Patch(ctx, namespace, ctrl.RawPatch(types.MergePatchType, patch))
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	r.logger.Info(fmt.Sprintf("ensured organization namespace %#q has organization labels", namespace.Name))

	return nil
}
