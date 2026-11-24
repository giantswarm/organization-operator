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
	// AgentPlatform is the organization's share of the installation's Agent
	// Platform. organization-operator does not read it.
	// +optional
	AgentPlatform *AgentPlatformSpec `json:"agentPlatform,omitempty"`
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
// organization whose spec.agentPlatform.workers.replicas is not set.
const DefaultAgentPlatformWorkerReplicas int32 = 1

// AgentPlatformWorkers sizes the organization's WorkerPool.
type AgentPlatformWorkers struct {
	// Replicas is the number of warm workers of the organization's WorkerPool
	// while the organization uses them: while its namespace holds a
	// ModelConfig and an Agent (a curated copy counts), or a ModelConfig and
	// a session. The pool runs 0 while the namespace holds no ModelConfig,
	// while it holds no Agent and no session, and while the organization is
	// Orphaned. A deleted organization runs Replicas until its sessions are
	// gone, so kagent can suspend its actors, and 0 after.
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
// one controller writes it: organization-operator writes Namespace, the Agent
// Platform's operator writes AgentPlatform. Each writer patches its own
// fields and keeps the others.
type OrganizationStatus struct {
	// Namespace is the namespace containing the resources for this organization.
	Namespace string `json:"namespace,omitempty"`

	// AgentPlatform is what the Agent Platform's operator observed for the
	// organization. organization-operator does not write it.
	// +optional
	AgentPlatform *AgentPlatformStatus `json:"agentPlatform,omitempty"`

	// Conditions are the conditions of every controller that reconciles the
	// organization, one for each type.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Erasure steps of an organization whose Agent Platform is erased, in order
// (status.agentPlatform.erasureStep).
const (
	AgentPlatformErasureWorkloads      = "Workloads"
	AgentPlatformErasureSessions       = "Sessions"
	AgentPlatformErasureSnapshots      = "Snapshots"
	AgentPlatformErasureNamespace      = "Namespace"
	AgentPlatformErasureOutsideObjects = "OutsideObjects"
)

// AgentPlatformStatus is what the Agent Platform's operator observed for the
// organization.
type AgentPlatformStatus struct {
	// Namespace is the namespace the Agent Platform keeps for the
	// organization: agents-<name>.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// SnapshotPrefix is where the organization's worker snapshots live,
	// s3://<bucket>/<uid>. Empty when the operator has no snapshot bucket.
	// +optional
	SnapshotPrefix string `json:"snapshotPrefix,omitempty"`

	// ErasureStep is the erasure step the operator is on once the
	// organization's Agent Platform is erased: Workloads, Sessions, Snapshots,
	// Namespace or OutsideObjects. Empty while it is not erased.
	// +optional
	// +kubebuilder:validation:Enum=Workloads;Sessions;Snapshots;Namespace;OutsideObjects
	ErasureStep string `json:"erasureStep,omitempty"`

	// SnapshotsErasedAt is when the operator first emptied the snapshot
	// prefix of the erased organization; it empties it once more after the
	// snapshot erase delay.
	// +optional
	SnapshotsErasedAt *metav1.MicroTime `json:"snapshotsErasedAt,omitempty"`

	// LastAppliedRevision is the last source revision the installation
	// Kustomization applied the organization from (e.g. <branch>@sha1:<sha>
	// for a GitRepository): that Kustomization's status.lastAppliedRevision
	// while the organization is in its settled inventory. It is kept while the
	// organization is Orphaned, so a removed file can be found in that
	// revision. Empty when no such revision was recorded.
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
