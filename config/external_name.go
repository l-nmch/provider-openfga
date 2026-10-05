package config

import (
	"context"
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/pkg/errors"
)

// ExternalNameConfigs contains all external name configurations for this
// provider. OpenFGA assigns a ULID to stores and authorization models, so both
// take their external name from the provider. Relationship tuples have no
// identifier at all: they are addressed by the tuple itself, and their external
// name is the composite key "terraform import" expects.
var ExternalNameConfigs = map[string]config.ExternalName{
	"openfga_store":               computedIdentifier(),
	"openfga_authorization_model": authorizationModel(),
	"openfga_relationship_tuple":  relationshipTuple(),
}

// Separator between the parts of a composite Terraform import ID. OpenFGA
// identifiers are ULIDs and tuple parts are "type:id" pairs, so none of them
// can contain a slash.
const separator = "/"

// placeholderID is a syntactically valid ULID that OpenFGA will never assign,
// so reading it answers "not found" rather than failing.
const placeholderID = "00000000000000000000000000"

// computedIdentifier identifies a resource by the ULID OpenFGA assigns it,
// which the "id" attribute holds, and stands in a placeholder for it until
// then.
//
// The placeholder is what makes creation work. Upjet refreshes a resource it
// has no state for from a state it builds out of the resource's parameters,
// with the external name as "id". Before creation that is empty, and this
// Terraform provider is built on the plugin framework, which has no
// convention that an empty ID means the resource is gone: it passes the empty
// string to OpenFGA, which rejects it ("storeId is required and must be
// specified") and fails the refresh instead of reporting the resource as
// absent. An ID that is well-formed but cannot exist gets a 404 instead, which
// the provider turns into an empty state, which is how upjet learns it has to
// create the resource.
func computedIdentifier() config.ExternalName {
	return config.NewExternalNameFrom(config.IdentifierFromProvider,
		config.WithGetIDFn(func(fn config.GetIDFn, ctx context.Context, externalName string, parameters map[string]any, cfg map[string]any) (string, error) {
			if externalName == "" {
				return placeholderID, nil
			}
			return fn(ctx, externalName, parameters, cfg)
		}),
		config.WithGetExternalNameFn(func(_ config.GetExternalNameFn, tfstate map[string]any) (string, error) {
			id, _ := tfstate["id"].(string)
			if id == "" || id == placeholderID {
				return "", errors.New("cannot find \"id\" in tfstate")
			}
			return id, nil
		}),
	)
}

// authorizationModel identifies an authorization model by its ULID, like any
// other computed identifier, but imports it by "<store_id>/<model_id>", which
// is the only form "terraform import" accepts. The two are separate hooks:
// GetIDFn fills the "id" attribute of the Terraform state, which holds the
// model ULID alone.
func authorizationModel() config.ExternalName {
	e := computedIdentifier()
	e.GetImportIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		storeID, err := stringParam(parameters, "store_id")
		if err != nil {
			return "", err
		}
		// This is built on every reconcile, including before the model
		// exists, so it falls back to the placeholder rather than failing.
		if externalName == "" {
			externalName = placeholderID
		}
		return storeID + separator + externalName, nil
	}
	return e
}

// relationshipTupleParts are the tuple attributes that identify a relationship
// tuple, in the order "terraform import" expects them after the store ID and
// the optional authorization model ID.
var relationshipTupleParts = []string{"user", "relation", "object"}

// relationshipTuple identifies a relationship tuple by
// "<store_id>[/<authorization_model_id>]/<user>/<relation>/<object>", which is
// both the only way to address it and the Terraform import ID. The resource
// has no "id" attribute, so the external name is read back from the tuple
// itself, and every part of it stays a regular spec field: nothing has to be
// set from the external name.
func relationshipTuple() config.ExternalName {
	return config.NewExternalNameFrom(config.IdentifierFromProvider,
		config.WithGetIDFn(func(_ config.GetIDFn, _ context.Context, _ string, parameters map[string]any, _ map[string]any) (string, error) {
			return tupleKey(parameters)
		}),
		config.WithGetExternalNameFn(func(_ config.GetExternalNameFn, tfstate map[string]any) (string, error) {
			return tupleKey(tfstate)
		}),
	)
}

// tupleKey builds the composite key of a relationship tuple from a map of
// Terraform attributes, which is either its parameters or its state.
//
// The authorization model is included when the tuple pins one. Importing
// without it leaves authorization_model_id unset in the state, which then
// differs from the spec and, since the field requires replacement, has
// Terraform plan a replacement; upjet reads a replacement plan as a resource
// that does not exist, so the tuple would be written again on every reconcile.
func tupleKey(attrs map[string]any) (string, error) {
	keys := []string{"store_id"}
	if m, _ := attrs["authorization_model_id"].(string); m != "" {
		keys = append(keys, "authorization_model_id")
	}
	keys = append(keys, relationshipTupleParts...)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v, err := stringParam(attrs, k)
		if err != nil {
			return "", err
		}
		if strings.Contains(v, separator) {
			return "", errors.Errorf("%q cannot contain %q", k, separator)
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, separator), nil
}

func stringParam(attrs map[string]any, name string) (string, error) {
	v, ok := attrs[name].(string)
	if !ok || v == "" {
		return "", errors.Errorf("cannot find %q in the resource attributes", name)
	}
	return v, nil
}

// ExternalNameConfigurations applies all external name configs listed in the
// table ExternalNameConfigs.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the list of all resources whose external name
// is configured manually.
func ExternalNameConfigured() []string {
	l := make([]string, len(ExternalNameConfigs))
	i := 0
	for name := range ExternalNameConfigs {
		// $ is added to match the exact string since the format is regex.
		l[i] = name + "$"
		i++
	}
	return l
}
