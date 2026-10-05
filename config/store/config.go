// Package store configures the OpenFGA store resource.
package store

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AnnotationKeyName records the name a store was created with, so that
// changing it afterwards can be rejected rather than acted upon.
const AnnotationKeyName = "openfga.crossplane.io/store-name"

// Configure configures the store group.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("openfga_store", func(r *config.Resource) {
		r.ShortGroup = "store"
		r.Kind = "Store"
		r.InitializerFns = append(r.InitializerFns, nameGuard)
	})
}

// nameGuard refuses to reconcile a store whose name changed since it was
// created. "name" is the only argument of openfga_store and the Terraform
// provider marks it as requiring replacement, which for a store means deleting
// it with every authorization model and relationship tuple it holds. Upjet
// reads a replacement plan as a resource that does not exist (it only looks at
// the "add" and "change" counts of the plan), so Crossplane would call Create
// and Terraform would destroy and recreate the store. Failing loudly is the
// only safe behaviour: to rename a store, create a new one.
func nameGuard(kube client.Client) managed.Initializer {
	return managed.InitializerFn(func(ctx context.Context, mg xpresource.Managed) error {
		// Until the store exists there is nothing to protect.
		if meta.GetExternalName(mg) == "" {
			return nil
		}
		tr, ok := mg.(resource.Terraformed)
		if !ok {
			return errors.New("managed resource is not Terraformed")
		}
		params, err := tr.GetParameters()
		if err != nil {
			return errors.Wrap(err, "cannot get parameters")
		}
		name, _ := params["name"].(string)
		if name == "" {
			return nil
		}

		created, ok := mg.GetAnnotations()[AnnotationKeyName]
		if !ok {
			// First reconcile of a store that exists, either just created or
			// adopted: its spec is the baseline to compare against later.
			meta.AddAnnotations(mg, map[string]string{AnnotationKeyName: name})
			return errors.Wrap(kube.Update(ctx, mg), "cannot record the store name")
		}
		if created != name {
			return errors.Errorf("the name of store %q cannot be changed to %q: OpenFGA has no rename operation and "+
				"applying it would delete the store with every authorization model and relationship tuple it holds; "+
				"create a new store instead, or reset spec.forProvider.name to %q", created, name, created)
		}
		return nil
	})
}
