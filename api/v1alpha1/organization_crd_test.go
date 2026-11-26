package v1alpha1_test

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	structurallisttype "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

const crdPath = "../../config/crd/bases/security.giantswarm.io_organizations.yaml"

// crdSchema is the schema of the generated CRD, the one the API server
// applies to an Organization.
type crdSchema struct {
	structural *structuralschema.Structural
	validator  validation.SchemaValidator
	rules      *cel.Validator
}

func loadCRDSchema(t *testing.T) crdSchema {
	t.Helper()

	raw, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("reading the CRD: %v", err)
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.UnmarshalStrict(raw, crd); err != nil {
		t.Fatalf("decoding the CRD: %v", err)
	}
	if len(crd.Spec.Versions) != 1 {
		t.Fatalf("the CRD has %d versions, want 1", len(crd.Spec.Versions))
	}

	// The API server refuses a CRD whose schema or validation rules it
	// cannot use, a rule over its cost limit included.
	internalCRD := &apiextensions.CustomResourceDefinition{}
	err = apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(
		crd, internalCRD, nil)
	if err != nil {
		t.Fatalf("converting the CRD: %v", err)
	}
	// The API server sets the stored versions when it creates the CRD.
	internalCRD.Status.StoredVersions = []string{internalCRD.Spec.Versions[0].Name}
	if errs := crdvalidation.ValidateCustomResourceDefinition(context.Background(), internalCRD); len(errs) > 0 {
		t.Fatalf("the API server refuses the CRD: %v", errs)
	}

	internal := &apiextensions.JSONSchemaProps{}
	err = apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
		crd.Spec.Versions[0].Schema.OpenAPIV3Schema, internal, nil)
	if err != nil {
		t.Fatalf("converting the CRD schema: %v", err)
	}
	structural, err := structuralschema.NewStructural(internal)
	if err != nil {
		t.Fatalf("building the structural schema: %v", err)
	}
	validator, _, err := validation.NewSchemaValidator(internal)
	if err != nil {
		t.Fatalf("building the schema validator: %v", err)
	}
	rules := cel.NewValidator(structural, true, celconfig.PerCallLimit)
	return crdSchema{structural: structural, validator: validator, rules: rules}
}

// admit does what the API server does to a new Organization: it drops the
// fields the schema does not know, applies the defaults and validates the
// rest, its validation rules (x-kubernetes-validations) included. It returns
// the dropped field paths and the validation errors.
func (s crdSchema) admit(obj map[string]interface{}) ([]string, field.ErrorList) {
	pruned := structuralpruning.PruneWithOptions(obj, s.structural, true,
		structuralschema.UnknownFieldPathOptions{TrackUnknownFieldPaths: true})
	structuraldefaulting.Default(obj, s.structural)
	errs := validation.ValidateCustomResource(nil, obj, s.validator)
	errs = append(errs, structurallisttype.ValidateListSetsAndMaps(nil, s.structural, obj)...)
	ruleErrs, _ := s.rules.Validate(context.Background(), nil, s.structural, obj, nil, celconfig.RuntimeCELCostBudget)
	errs = append(errs, ruleErrs...)
	return pruned, errs
}

func organizationFromYAML(t *testing.T, doc string) map[string]interface{} {
	t.Helper()

	obj := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
		t.Fatalf("decoding the Organization: %v", err)
	}
	return obj
}

func TestOrganizationCRDAdmission(t *testing.T) {
	schema := loadCRDSchema(t)

	testCases := []struct {
		name string
		doc  string
		// wantErrorPath is the field the API server refuses; empty when it
		// admits the object.
		wantErrorPath string
		// wantDropped are the fields the API server does not know: it drops
		// them, and refuses the object when the writer asks for strict field
		// validation (kubectl's default).
		wantDropped []string
		// wantSpec is the spec the API server stores, after its defaults.
		wantSpec string
	}{
		{
			name: "an empty spec stays valid and gets no Agent Platform",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
`,
			wantSpec: `{}`,
		},
		{
			name: "an agentPlatform block without workers gets one replica",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  capabilities:
    agentPlatform: {enabled: true, harnesses: [claude-code, codex]}
`,
			wantSpec: `{"capabilities":{"agentPlatform":` +
				`{"enabled":true,"harnesses":["claude-code","codex"],"workers":{"replicas":1}}}}`,
		},
		{
			name: "zero replicas set on purpose stay zero",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  capabilities:
    agentPlatform: {enabled: true, workers: {replicas: 0}}
`,
			wantSpec: `{"capabilities":{"agentPlatform":{"enabled":true,"workers":{"replicas":0}}}}`,
		},
		{
			name: "negative replicas are refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  capabilities:
    agentPlatform: {enabled: true, workers: {replicas: -1}}
`,
			wantErrorPath: "spec.capabilities.agentPlatform.workers.replicas",
		},
		{
			name: "a harness named twice is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  capabilities:
    agentPlatform: {enabled: true, harnesses: [claude-code, claude-code]}
`,
			wantErrorPath: "spec.capabilities.agentPlatform.harnesses[1]",
		},
		{
			name: "the Agent Platform fields outside capabilities are not part of the schema",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  agentPlatform: {enabled: true}
status:
  agentPlatform: {namespace: agents-acme}
`,
			wantDropped: []string{"spec.agentPlatform", "status.agentPlatform"},
			wantSpec:    `{}`,
		},
		{
			name: "users and groups are admins, and a user and a group may share a name",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  admins:
  - {kind: Group, name: acme-admins}
  - {kind: User, name: alice@example.com}
  - {kind: User, name: acme-admins}
`,
			wantSpec: `{"admins":[` +
				`{"kind":"Group","name":"acme-admins"},` +
				`{"kind":"User","name":"alice@example.com"},` +
				`{"kind":"User","name":"acme-admins"}]}`,
		},
		{
			name: "a ServiceAccount admin is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  admins:
  - {kind: ServiceAccount, name: deployer}
`,
			wantErrorPath: "spec.admins[0].kind",
		},
		{
			name: "an admin with an empty name is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  admins:
  - {kind: Group, name: ""}
`,
			wantErrorPath: "spec.admins[0].name",
		},
		{
			name: "an admin whose name starts with system: is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  admins:
  - {kind: Group, name: acme-admins}
  - {kind: Group, name: "system:authenticated"}
`,
			wantErrorPath: "spec.admins[1].name",
		},
		{
			name: "an admin named twice is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  admins:
  - {kind: Group, name: acme-admins}
  - {kind: Group, name: acme-admins}
`,
			wantErrorPath: "spec.admins[1]",
		},
		{
			name: "a namespace without a capability is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
status:
  namespaces:
  - {name: org-acme}
`,
			wantErrorPath: "status.namespaces[0].capability",
		},
		{
			name: "a namespace without a name is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
status:
  namespaces:
  - {capability: kaas}
`,
			wantErrorPath: "status.namespaces[0].name",
		},
		{
			name: "two namespaces of one capability are refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
status:
  namespaces:
  - {name: agents-acme, capability: agentPlatform}
  - {name: agents-acme-2, capability: agentPlatform}
`,
			wantErrorPath: "status.namespaces[1]",
		},
		{
			name: "two conditions of one type are refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
status:
  conditions:
  - {type: Ready, status: "True", reason: Reconciled, message: "", lastTransitionTime: "2026-11-24T10:00:00Z"}
  - {type: Ready, status: "False", reason: Failed, message: "", lastTransitionTime: "2026-11-24T10:00:00Z"}
`,
			wantErrorPath: "status.conditions[1]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj := organizationFromYAML(t, tc.doc)

			dropped, errs := schema.admit(obj)

			if tc.wantErrorPath != "" {
				for _, err := range errs {
					if err.Field == tc.wantErrorPath {
						return
					}
				}
				t.Fatalf("want an error on %s, got %v", tc.wantErrorPath, errs)
			}
			if len(errs) > 0 {
				t.Fatalf("want the object admitted, got %v", errs)
			}
			slices.Sort(dropped)
			if !slices.Equal(dropped, tc.wantDropped) {
				t.Fatalf("dropped fields are %v, want %v", dropped, tc.wantDropped)
			}
			gotSpec, err := json.Marshal(obj["spec"])
			if err != nil {
				t.Fatalf("encoding the spec: %v", err)
			}
			if string(gotSpec) != tc.wantSpec {
				t.Fatalf("stored spec is %s, want %s", gotSpec, tc.wantSpec)
			}
		})
	}
}

// TestOrganizationCRDKeepsEveryGoField guards the agreement of the Go types
// and the generated CRD: a field of the Go types that the CRD does not know
// is dropped by the API server, so a writer of that field loses it.
func TestOrganizationCRDKeepsEveryGoField(t *testing.T) {
	schema := loadCRDSchema(t)

	org := securityv1alpha1.Organization{
		TypeMeta:   metav1.TypeMeta{APIVersion: "security.giantswarm.io/v1alpha1", Kind: "Organization"},
		ObjectMeta: metav1.ObjectMeta{Name: "acme"},
		Spec: securityv1alpha1.OrganizationSpec{
			Admins: []securityv1alpha1.OrganizationAdmin{
				{Kind: securityv1alpha1.OrganizationAdminGroup, Name: "acme-admins"},
				{Kind: securityv1alpha1.OrganizationAdminUser, Name: "alice@example.com"},
			},
			Capabilities: securityv1alpha1.OrganizationCapabilities{
				AgentPlatform: &securityv1alpha1.AgentPlatformSpec{
					Enabled:   true,
					Harnesses: []string{"claude-code"},
					Workers:   securityv1alpha1.AgentPlatformWorkers{Replicas: ptr.To[int32](2)},
				},
			},
		},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: "org-acme",
			Namespaces: []securityv1alpha1.OrganizationNamespace{
				{Name: "org-acme", Capability: securityv1alpha1.CapabilityKaaS},
				{Name: "agents-acme", Capability: securityv1alpha1.CapabilityAgentPlatform},
			},
			Capabilities: securityv1alpha1.OrganizationCapabilitiesStatus{
				AgentPlatform: &securityv1alpha1.AgentPlatformStatus{
					SnapshotPrefix:      "s3://snapshots/0b6f",
					LastAppliedRevision: "main@sha1:4fb1b22",
				},
			},
			Conditions: []metav1.Condition{{
				Type:               securityv1alpha1.AgentPlatformErasing,
				Status:             metav1.ConditionTrue,
				Reason:             securityv1alpha1.AgentPlatformErasingSnapshots,
				LastTransitionTime: metav1.Now().Rfc3339Copy(),
			}},
		},
	}
	raw, err := json.Marshal(org)
	if err != nil {
		t.Fatalf("encoding the Organization: %v", err)
	}
	obj := map[string]interface{}{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("decoding the Organization: %v", err)
	}

	pruned, errs := schema.admit(obj)

	if len(pruned) > 0 {
		t.Fatalf("the CRD drops fields of the Go types: %s", strings.Join(pruned, ", "))
	}
	if len(errs) > 0 {
		t.Fatalf("the CRD refuses an Organization built from the Go types: %v", errs)
	}
}

func TestAgentPlatformWorkersReplicasOrDefault(t *testing.T) {
	testCases := []struct {
		name     string
		replicas *int32
		want     int32
	}{
		{name: "not set means one", replicas: nil, want: 1},
		{name: "zero set on purpose stays zero", replicas: ptr.To[int32](0), want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := securityv1alpha1.AgentPlatformWorkers{Replicas: tc.replicas}.ReplicasOrDefault()
			if got != tc.want {
				t.Fatalf("ReplicasOrDefault() = %d, want %d", got, tc.want)
			}
		})
	}
}
