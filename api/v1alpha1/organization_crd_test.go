package v1alpha1_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	structurallisttype "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
	return crdSchema{structural: structural, validator: validator}
}

// admit does what the API server does to a new Organization: it drops the
// fields the schema does not know, applies the defaults and validates the
// rest. It returns the dropped field paths and the validation errors.
func (s crdSchema) admit(obj map[string]interface{}) ([]string, field.ErrorList) {
	pruned := structuralpruning.PruneWithOptions(obj, s.structural, true,
		structuralschema.UnknownFieldPathOptions{TrackUnknownFieldPaths: true})
	structuraldefaulting.Default(obj, s.structural)
	errs := validation.ValidateCustomResource(nil, obj, s.validator)
	errs = append(errs, structurallisttype.ValidateListSetsAndMaps(nil, s.structural, obj)...)
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
  agentPlatform: {enabled: true, harnesses: [claude-code, codex]}
`,
			wantSpec: `{"agentPlatform":{"enabled":true,"harnesses":["claude-code","codex"],"workers":{"replicas":1}}}`,
		},
		{
			name: "zero replicas set on purpose stay zero",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  agentPlatform: {enabled: true, workers: {replicas: 0}}
`,
			wantSpec: `{"agentPlatform":{"enabled":true,"workers":{"replicas":0}}}`,
		},
		{
			name: "negative replicas are refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  agentPlatform: {enabled: true, workers: {replicas: -1}}
`,
			wantErrorPath: "spec.agentPlatform.workers.replicas",
		},
		{
			name: "a harness named twice is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec:
  agentPlatform: {enabled: true, harnesses: [claude-code, claude-code]}
`,
			wantErrorPath: "spec.agentPlatform.harnesses[1]",
		},
		{
			name: "an unknown erasure step is refused",
			doc: `
apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata: {name: acme}
spec: {}
status:
  agentPlatform: {erasureStep: Everything}
`,
			wantErrorPath: "status.agentPlatform.erasureStep",
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

			_, errs := schema.admit(obj)

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

	erasedAt := metav1.NewMicroTime(metav1.Now().Rfc3339Copy().Time)
	org := securityv1alpha1.Organization{
		TypeMeta:   metav1.TypeMeta{APIVersion: "security.giantswarm.io/v1alpha1", Kind: "Organization"},
		ObjectMeta: metav1.ObjectMeta{Name: "acme"},
		Spec: securityv1alpha1.OrganizationSpec{
			AgentPlatform: &securityv1alpha1.AgentPlatformSpec{
				Enabled:   true,
				Harnesses: []string{"claude-code"},
				Workers:   securityv1alpha1.AgentPlatformWorkers{Replicas: ptr.To[int32](2)},
			},
		},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: "org-acme",
			AgentPlatform: &securityv1alpha1.AgentPlatformStatus{
				Namespace:           "agents-acme",
				SnapshotPrefix:      "s3://snapshots/0b6f",
				ErasureStep:         securityv1alpha1.AgentPlatformErasureSnapshots,
				SnapshotsErasedAt:   &erasedAt,
				LastAppliedRevision: "main@sha1:4fb1b22",
			},
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "Reconciled",
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
