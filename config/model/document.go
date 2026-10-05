package model

import (
	"bytes"
	"encoding/json"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	openfgasdk "github.com/openfga/go-sdk"
	"github.com/openfga/language/pkg/go/transformer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

// document is an authorization model as the Terraform provider serialises it
// into model_json, which is also how it reads it back from OpenFGA. Matching
// it exactly matters: the provider compares model_json semantically, so a key
// we emit and it does not (every zero value protojson writes out, for
// instance) reads as a difference. Since every argument of the resource
// requires replacement, and upjet sees a replacement plan as a resource that
// does not exist, such a difference would have each reconcile write a new
// version of the model, forever.
//
// This mirrors AuthorizationModelWithoutId in the Terraform provider.
type document struct {
	openfgasdk.AuthorizationModel
	ID *any `json:"-"`
}

// MarshalJSON implements json.Marshaler.
func (d document) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"schema_version":   d.SchemaVersion,
		"type_definitions": d.TypeDefinitions,
	}
	if d.Conditions != nil {
		out["conditions"] = d.Conditions
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	// As upstream does, so that condition expressions stay readable.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// canonicalJSON renders an authorization model into the model_json the
// Terraform provider takes: marshal the protobuf model with protojson, read it
// back into the document shape, which drops the server-assigned id, and render
// that. This is what the provider's own openfga_authorization_model_document
// data source does, and the docs tell you to feed that data source into
// model_json.
//
// EmitDefaultValues is what makes it match. The provider replaces model_json
// in the state with what it marshals from the model it reads back, where
// OpenFGA has filled in every unset field, so the state holds "module": "",
// "conditions": {} and the like. Leaving them out of the configuration reads
// as a difference, and since every argument of the resource requires
// replacement, and upjet sees a replacement plan as a resource that does not
// exist, each reconcile would then write another version of the model.
// Verified against OpenFGA: with defaults emitted, a second plan is empty.
func canonicalJSON(m *openfgav1.AuthorizationModel) (string, error) {
	b, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(m)
	if err != nil {
		return "", errors.Wrap(err, "cannot marshal the authorization model")
	}
	var doc document
	if err := json.Unmarshal(b, &doc); err != nil {
		return "", errors.Wrap(err, "cannot read back the marshalled authorization model")
	}
	// json.Marshal compacts what MarshalJSON returns, dropping the newline
	// json.Encoder appends.
	out, err := json.Marshal(doc)
	if err != nil {
		return "", errors.Wrap(err, "cannot render the authorization model document")
	}
	return string(out), nil
}

// FromDSL turns an OpenFGA DSL model into the canonical JSON document the
// Terraform provider's model_json argument takes.
func FromDSL(dsl string) (string, error) {
	m, err := transformer.TransformDSLToProto(dsl)
	if err != nil {
		return "", errors.Wrap(err, "cannot parse the authorization model DSL")
	}
	return canonicalJSON(m)
}

// FromJSON validates an authorization model given as JSON and renders it in
// the same canonical form as FromDSL, so that a model reaches OpenFGA
// identically however it was written.
func FromJSON(doc string) (string, error) {
	m, err := transformer.LoadJSONStringToProto(doc)
	if err != nil {
		return "", errors.Wrap(err, "cannot parse the authorization model JSON")
	}
	return canonicalJSON(m)
}
