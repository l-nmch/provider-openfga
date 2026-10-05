// Package model configures the OpenFGA authorization model resource.
package model

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// argJSON is the only way the Terraform provider accepts a model.
	argJSON = "model_json"
	// argDSL is added by this provider: the OpenFGA DSL, which is what models
	// are actually written in. The Terraform provider only converts it through
	// a data source, and upjet generates managed resources from resources
	// only, so the conversion happens here instead.
	argDSL = "model_dsl"
)

// Configure configures the model group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("openfga_authorization_model", func(r *config.Resource) {
		r.ShortGroup = "model"
		r.Kind = "AuthorizationModel"
		r.References["store_id"] = config.Reference{
			TerraformName: "openfga_store",
		}

		// Expose modelDsl next to modelJson, and make modelJson optional so
		// that either one identifies the model. Exactly one of them is
		// required, which dslValidator enforces.
		r.TerraformResource.Schema[argDSL] = &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Description: "The authorization model definition in the OpenFGA DSL. " +
				"Converted to the canonical JSON form before it is applied. " +
				"Exactly one of modelDsl and modelJson must be set.",
		}
		if s, ok := r.TerraformResource.Schema[argJSON]; ok {
			s.Required = false
			s.Optional = true
		}

		r.TerraformConversions = append(r.TerraformConversions, documentConversion{})
		// The canonical form of the model is what reaches OpenFGA, not what
		// the user wrote, so it must not be copied back over the spec.
		r.LateInitializer.IgnoredFields = append(r.LateInitializer.IgnoredFields, argJSON)
		r.InitializerFns = append(r.InitializerFns, dslValidator)
	})
}

// document renders the model into the canonical JSON the Terraform provider
// takes, from either argDSL or argJSON, and removes argDSL, which is not part
// of the Terraform schema and would be rejected as an unknown argument.
type documentConversion struct{}

// Convert implements config.TerraformConversion.
func (documentConversion) Convert(params map[string]any, _ *config.Resource, mode config.Mode) (map[string]any, error) {
	if mode != config.ToTerraform {
		return params, nil
	}
	dsl, _ := params[argDSL].(string)
	doc, _ := params[argJSON].(string)
	delete(params, argDSL)

	switch {
	case dsl != "" && doc != "":
		return nil, errors.Errorf("only one of %q and %q may be set", argDSL, argJSON)
	case dsl != "":
		rendered, err := FromDSL(dsl)
		if err != nil {
			return nil, err
		}
		params[argJSON] = rendered
	case doc != "":
		rendered, err := FromJSON(doc)
		if err != nil {
			return nil, err
		}
		params[argJSON] = rendered
	}
	// Neither is set: this map is an observation rather than a set of
	// parameters, or an invalid spec dslValidator already rejected.
	return params, nil
}

// dslValidator rejects a model that sets neither modelDsl nor modelJson, or
// both. The Terraform schema cannot express "exactly one of", since both
// arguments have to be optional for either to be usable on its own.
func dslValidator(_ client.Client) managed.Initializer {
	return managed.InitializerFn(func(_ context.Context, mg xpresource.Managed) error {
		tr, ok := mg.(resource.Terraformed)
		if !ok {
			return errors.New("managed resource is not Terraformed")
		}
		params, err := tr.GetParameters()
		if err != nil {
			return errors.Wrap(err, "cannot get parameters")
		}
		dsl, _ := params[argDSL].(string)
		doc, _ := params[argJSON].(string)
		switch {
		case dsl == "" && doc == "":
			return errors.Errorf("one of %q and %q must be set", argDSL, argJSON)
		case dsl != "" && doc != "":
			return errors.Errorf("only one of %q and %q may be set", argDSL, argJSON)
		}
		return nil
	})
}
