// Package tuple configures the OpenFGA relationship tuple resource.
package tuple

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the tuple group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("openfga_relationship_tuple", func(r *config.Resource) {
		r.ShortGroup = "tuple"
		r.Kind = "RelationshipTuple"
		// The external name of a Store and of an AuthorizationModel is the
		// ULID OpenFGA assigned to it, which is exactly what these two
		// arguments take, so the default extractor applies.
		r.References["store_id"] = config.Reference{
			TerraformName: "openfga_store",
		}
		r.References["authorization_model_id"] = config.Reference{
			TerraformName: "openfga_authorization_model",
		}
	})
}
