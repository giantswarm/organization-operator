package service

import (
	"github.com/giantswarm/operatorkit/v8/pkg/flag/service/kubernetes"

	"github.com/giantswarm/organization-operator/flag/service/legacycredentials"
	"github.com/giantswarm/organization-operator/flag/service/legacyorganizations"
)

// Service is an intermediate data structure for command line configuration flags.
type Service struct {
	Kubernetes          kubernetes.Kubernetes
	LegacyOrganizations legacyorganizations.LegacyOrganizations
	LegacyCredentials   legacycredentials.LegacyCredentials
	ResyncPeriod        string
}
