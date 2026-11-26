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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	// Admins are the Users and Groups the organization is handed over to.
	// Each product derives its admin binding in the organization from this
	// list. No admins means the platform team manages the organization.
	// organization-operator does not read it.
	// +optional
	// +listType=map
	// +listMapKey=kind
	// +listMapKey=name
	Admins []OrganizationAdmin `json:"admins,omitempty"`

	// Capabilities are the settings of the platform capabilities the
	// organization gets, one field for each capability.
	// +optional
	Capabilities OrganizationCapabilities `json:"capabilities,omitzero"`
}

// OrganizationCapabilities are the settings of the platform capabilities the
// organization gets. The operator of each capability reads its own field.
type OrganizationCapabilities struct {
	// AgentPlatform is the organization's share of the installation's Agent
	// Platform. organization-operator does not read it.
	// +optional
	AgentPlatform *AgentPlatformSpec `json:"agentPlatform,omitempty"`
}

// Capabilities that give an organization a namespace
// (status.namespaces[].capability).
const (
	// CapabilityKaaS is the organization's namespace org-<name>, which
	// organization-operator creates.
	CapabilityKaaS = "kaas"
	// CapabilityAgentPlatform is the organization's namespace on the Agent
	// Platform, which the Agent Platform's operator creates.
	CapabilityAgentPlatform = "agentPlatform"
)

// Kinds of an organization admin (spec.admins[].kind). They are the RBAC
// subject kinds of the same names.
const (
	OrganizationAdminUser  = "User"
	OrganizationAdminGroup = "Group"
)

// OrganizationAdmin is one admin of the organization: a User or a Group, by
// the name the API server authenticates it with. A ServiceAccount is not an
// admin, and neither is a name of the API server's own identities (system:).
type OrganizationAdmin struct {
	// Kind is User or Group.
	// +kubebuilder:validation:Enum=User;Group
	Kind string `json:"kind"`

	// Name is the user's or the group's name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="!self.startsWith('system:')",message="a system: name is not an admin"
	Name string `json:"name"`
}

// AgentPlatformSpec is what the organization gets on the Agent Platform.
type AgentPlatformSpec struct {
	// Enabled gives the organization the Agent Platform and its namespace
	// there.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Harnesses are the names of the curated Harnesses the organization gets.
	// +optional
	// +listType=set
	Harnesses []string `json:"harnesses,omitempty"`

	// Workers sizes the organization's WorkerPool.
	// +optional
	// +kubebuilder:default={}
	Workers AgentPlatformWorkers `json:"workers,omitempty"`
}

// DefaultAgentPlatformWorkerReplicas is the number of warm workers of an
// organization whose spec.capabilities.agentPlatform.workers.replicas is not
// set.
const DefaultAgentPlatformWorkerReplicas int32 = 1

// AgentPlatformWorkers sizes the organization's WorkerPool.
type AgentPlatformWorkers struct {
	// Replicas is the number of warm workers of the organization's WorkerPool
	// while the organization uses them: while its namespace holds a
	// ModelConfig and an Agent (a curated copy counts), or a ModelConfig and
	// a session. The pool runs 0 while the namespace holds no ModelConfig,
	// while it holds no Agent and no session, and while the organization's
	// AgentPlatformOrphaned condition is True. A deleted organization runs
	// Replicas until its sessions are gone, so kagent can suspend its actors,
	// and 0 after.
	//
	// Not set means 1; 0 set on purpose runs no worker.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`
}

// ReplicasOrDefault is Replicas, or DefaultAgentPlatformWorkerReplicas when it
// is not set. The CRD writes that default into an agentPlatform block without
// the field; a reader applies it too, for an object read from an API server
// whose CRD is older.
func (w AgentPlatformWorkers) ReplicasOrDefault() int32 {
	if w.Replicas == nil {
		return DefaultAgentPlatformWorkerReplicas
	}
	return *w.Replicas
}

// OrganizationStatus defines the observed state of Organization. More than
// one controller writes it: organization-operator writes Namespace and its
// kaas entry of Namespaces, the Agent Platform's operator writes its
// agentPlatform entry of Namespaces, Capabilities.AgentPlatform and its
// AgentPlatform conditions. Each writer writes its own fields and entries and
// keeps the others. A JSON merge patch replaces a whole list, so a writer of
// Namespaces or Conditions uses server-side apply with a field manager of its
// own, or a patch with an optimistic lock.
type OrganizationStatus struct {
	// Namespace is the namespace containing the resources for this organization.
	Namespace string `json:"namespace,omitempty"`

	// Namespaces are the organization's namespaces, one for each capability
	// that gives the organization a namespace. The operator of each
	// capability writes its own entry: the kaas entry is Namespace. Read a
	// capability's namespace from here; do not derive it from the name.
	// +optional
	// +listType=map
	// +listMapKey=capability
	Namespaces []OrganizationNamespace `json:"namespaces,omitempty"`

	// Capabilities are what the operator of each capability observed for the
	// organization, one field for each capability.
	// +optional
	Capabilities OrganizationCapabilitiesStatus `json:"capabilities,omitzero"`

	// Conditions are the conditions of every controller that reconciles the
	// organization, one for each type. The type of a capability's condition
	// starts with the capability's name in upper camel case, for example
	// AgentPlatformReady.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// OrganizationNamespace is a namespace of the organization and the capability
// it belongs to.
type OrganizationNamespace struct {
	// Name is the namespace's name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Capability is the capability that the namespace belongs to, for example
	// kaas or agentPlatform.
	// +kubebuilder:validation:MinLength=1
	Capability string `json:"capability"`
}

// OrganizationCapabilitiesStatus are what the operator of each capability
// observed for the organization.
type OrganizationCapabilitiesStatus struct {
	// AgentPlatform is what the Agent Platform's operator observed for the
	// organization. organization-operator does not write it.
	// +optional
	AgentPlatform *AgentPlatformStatus `json:"agentPlatform,omitempty"`
}

// Condition types of the Agent Platform (status.conditions[].type). The Agent
// Platform's operator writes them.
const (
	// AgentPlatformReady is True when the organization's Agent Platform
	// namespace holds every object the operator keeps there.
	AgentPlatformReady = "AgentPlatformReady"
	// AgentPlatformOrphaned is True when the organization carries the
	// installation Kustomization's ownership labels but is missing from that
	// Kustomization's inventory. Its reasons are AgentPlatformOrphaned*.
	AgentPlatformOrphaned = "AgentPlatformOrphaned"
	// AgentPlatformDefaultModelConfigured is False while the organization's
	// Agent Platform namespace holds no default ModelConfig.
	AgentPlatformDefaultModelConfigured = "AgentPlatformDefaultModelConfigured"
	// AgentPlatformErasing is True while the operator erases the
	// organization's Agent Platform. Its reason is the erasure step,
	// AgentPlatformErasing*.
	AgentPlatformErasing = "AgentPlatformErasing"
	// AgentPlatformSnapshotsErased is True from the operator's first delete of
	// the organization's snapshot prefix; its lastTransitionTime is the time
	// of that delete.
	AgentPlatformSnapshotsErased = "AgentPlatformSnapshotsErased"
)

// Reasons of the AgentPlatformOrphaned condition.
const (
	AgentPlatformOrphanedInInventory    = "InInventory"
	AgentPlatformOrphanedMissingFromGit = "MissingFromGit"
	AgentPlatformOrphanedDeleteMerged   = "DeleteMerged"
)

// Reasons of the AgentPlatformErasing condition: the erasure steps, in order.
const (
	AgentPlatformErasingWorkloads      = "Workloads"
	AgentPlatformErasingSessions       = "Sessions"
	AgentPlatformErasingSnapshots      = "Snapshots"
	AgentPlatformErasingNamespace      = "Namespace"
	AgentPlatformErasingOutsideObjects = "OutsideObjects"
)

// AgentPlatformStatus is what the Agent Platform's operator observed for the
// organization.
type AgentPlatformStatus struct {
	// SnapshotPrefix is where the organization's worker snapshots live,
	// s3://<bucket>/<uid>. The erasure deletes the prefix recorded here. Empty
	// when the operator has no snapshot bucket.
	// +optional
	SnapshotPrefix string `json:"snapshotPrefix,omitempty"`

	// LastAppliedRevision is the last source revision the installation
	// Kustomization applied the organization from (e.g. <branch>@sha1:<sha>
	// for a GitRepository): that Kustomization's status.lastAppliedRevision
	// while the organization is in its settled inventory. It is kept while the
	// organization's AgentPlatformOrphaned condition is True, so a removed file
	// can be found in that revision. Empty when no such revision was recorded.
	// +optional
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`
}

//nolint:revive
//+kubebuilder:object:root=true
//nolint:revive
//+kubebuilder:subresource:status
//nolint:revive
//+kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".status.namespace"
//nolint:revive
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//nolint:revive
//+kubebuilder:resource:scope=Cluster,categories={common,giantswarm},shortName={org,orgs}

// Organization represents schema for managed Kubernetes namespace.
// Reconciled by organization-operator.
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

//nolint:revive
//+kubebuilder:object:root=true

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
