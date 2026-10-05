package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/google/go-cmp/cmp"
)

const dsl = `model
  schema 1.1

type user

type document
  relations
    define viewer: [user]
`

func TestFromDSL(t *testing.T) {
	got, err := FromDSL(dsl)
	if err != nil {
		t.Fatalf("FromDSL: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatalf("the rendered document is not JSON: %v", err)
	}
	// An authorization model document holds the schema version and the type
	// definitions, never the server-assigned id.
	if _, ok := doc["id"]; ok {
		t.Error("the rendered document carries an id")
	}
	if v := doc["schema_version"]; v != "1.1" {
		t.Errorf("schema_version: got %v, want 1.1", v)
	}
	types, ok := doc["type_definitions"].([]any)
	if !ok || len(types) != 2 {
		t.Fatalf("type_definitions: got %v, want 2 entries", doc["type_definitions"])
	}
}

func TestFromJSONIsCanonical(t *testing.T) {
	canonical, err := FromDSL(dsl)
	if err != nil {
		t.Fatalf("FromDSL: %v", err)
	}

	// Whatever shape the same model is written in, it must reach OpenFGA
	// identically, so that the spec does not drift from the state.
	for name, in := range map[string]string{
		"AlreadyCanonical": canonical,
		"Reindented":       reindent(t, canonical),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := FromJSON(in)
			if err != nil {
				t.Fatalf("FromJSON: %v", err)
			}
			if diff := cmp.Diff(canonical, got); diff != "" {
				t.Errorf("not canonical: %s", diff)
			}
		})
	}
}

// reindent re-renders a JSON document with indentation and reordered keys, so
// that only its meaning is preserved.
func reindent(t *testing.T, doc string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(doc), &v); err != nil {
		t.Fatalf("cannot parse %q: %v", doc, err)
	}
	out, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		t.Fatalf("cannot re-render: %v", err)
	}
	return string(out)
}

func TestFromDSLErrors(t *testing.T) {
	// Only the syntax is checked here, as in the Terraform provider's own
	// authorization_model_document data source: a model that parses but makes
	// no sense, such as one relating a type that is not defined, is rejected by
	// OpenFGA when it is written.
	for name, in := range map[string]string{
		"Empty":         "",
		"NotADSL":       "{\"schema_version\": \"1.1\"}",
		"MissingSchema": "type user\n",
		"BadRelation":   "model\n  schema 1.1\n\ntype user\n\ntype document\n  relations\n    define viewer\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := FromDSL(in); err == nil {
				t.Errorf("expected an error, got %q", got)
			}
		})
	}
}

func TestFromJSONErrors(t *testing.T) {
	for name, in := range map[string]string{
		"Empty":      "",
		"NotJSON":    "model\n  schema 1.1\n",
		"BrokenJSON": "{\"schema_version\":",
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := FromJSON(in); err == nil {
				t.Errorf("expected an error, got %q", got)
			}
		})
	}
}

func TestDocumentConvert(t *testing.T) {
	canonical, err := FromDSL(dsl)
	if err != nil {
		t.Fatalf("FromDSL: %v", err)
	}

	t.Run("DSLBecomesJSON", func(t *testing.T) {
		got, err := (documentConversion{}).Convert(map[string]any{
			"store_id": "01K9B724V3EGG6B3R99R6642EQ",
			argDSL:     dsl,
		}, nil, config.ToTerraform)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		// model_dsl is not part of the Terraform schema: leaving it in would
		// make Terraform reject the configuration as an unknown argument.
		if _, ok := got[argDSL]; ok {
			t.Error("model_dsl reached Terraform")
		}
		if diff := cmp.Diff(canonical, got[argJSON]); diff != "" {
			t.Errorf("model_json: %s", diff)
		}
	})

	t.Run("JSONIsNormalised", func(t *testing.T) {
		got, err := (documentConversion{}).Convert(map[string]any{
			argJSON: reindent(t, canonical),
		}, nil, config.ToTerraform)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		if diff := cmp.Diff(canonical, got[argJSON]); diff != "" {
			t.Errorf("model_json: %s", diff)
		}
	})

	t.Run("BothIsAnError", func(t *testing.T) {
		if _, err := (documentConversion{}).Convert(map[string]any{
			argDSL:  dsl,
			argJSON: canonical,
		}, nil, config.ToTerraform); err == nil {
			t.Error("expected an error, got none")
		}
	})

	t.Run("NeitherIsLeftAlone", func(t *testing.T) {
		// Convert also runs over the observation, which carries neither, so it
		// must not fail there. An empty spec is rejected by dslValidator.
		in := map[string]any{"id": "01JF0PT3CRW2HAHBK4Z4RJBEWQ"}
		got, err := (documentConversion{}).Convert(in, nil, config.ToTerraform)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		if diff := cmp.Diff(in, got); diff != "" {
			t.Errorf("observation changed: %s", diff)
		}
	})

	t.Run("FromTerraformIsUntouched", func(t *testing.T) {
		in := map[string]any{argJSON: reindent(t, canonical)}
		got, err := (documentConversion{}).Convert(in, nil, config.FromTerraform)
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		if diff := cmp.Diff(in, got); diff != "" {
			t.Errorf("state changed: %s", diff)
		}
	})

	t.Run("InvalidDSLFails", func(t *testing.T) {
		if _, err := (documentConversion{}).Convert(map[string]any{argDSL: "not a model"}, nil, config.ToTerraform); err == nil {
			t.Error("expected an error, got none")
		} else if !strings.Contains(err.Error(), "DSL") {
			t.Errorf("unhelpful error: %v", err)
		}
	})
}

// TestCanonicalJSONMatchesProviderShape pins the serialisation down to the
// shape the Terraform provider puts in the state when it reads a model back,
// where OpenFGA has filled in every unset field. Anything else reads as a
// difference and, because every argument of the resource requires replacement,
// would have each reconcile write a new version of the model. Checked against
// a real OpenFGA server: with this exact form, a second plan is empty.
func TestCanonicalJSONMatchesProviderShape(t *testing.T) {
	got, err := FromDSL(dsl)
	if err != nil {
		t.Fatalf("FromDSL: %v", err)
	}
	want := `{"conditions":{},"schema_version":"1.1","type_definitions":` +
		`[{"relations":{},"type":"user"},` +
		`{"metadata":{"module":"","relations":{"viewer":{"directly_related_user_types":` +
		`[{"condition":"","type":"user"}],"module":""}}},` +
		`"relations":{"viewer":{"this":{}}},"type":"document"}]}`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("canonical JSON: %s", diff)
	}
	// The server-assigned id is the one thing that must never be in there.
	if strings.Contains(got, `"id":`) {
		t.Errorf("canonical JSON carries an id: %s", got)
	}
}

// TestCanonicalJSONIsStable checks a model with a union and a condition, the
// shapes most likely to serialise differently from what OpenFGA returns.
func TestCanonicalJSONIsStable(t *testing.T) {
	const withUnion = `model
  schema 1.1

type user

type group
  relations
    define member: [user]

type project
  relations
    define admin: [user, group#member]
    define viewer: [user, group#member] or admin
`
	got, err := FromDSL(withUnion)
	if err != nil {
		t.Fatalf("FromDSL: %v", err)
	}
	// Every optional field OpenFGA fills in has to be present, including the
	// ones nested inside a union.
	for _, want := range []string{
		`"conditions":{}`,
		`"module":""`,
		`"condition":""`,
		`"computedUserset":{"object":"","relation":"admin"}`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("canonical JSON is missing %s: %s", want, got)
		}
	}
	// And it must survive a round trip through the JSON entry point.
	again, err := FromJSON(got)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}
	if diff := cmp.Diff(got, again); diff != "" {
		t.Errorf("not stable through FromJSON: %s", diff)
	}
}
