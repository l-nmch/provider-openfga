# v0.1.0

First release. All three resources of terraform-provider-openfga v0.5.1
(`Store`, `AuthorizationModel`, `RelationshipTuple`), namespaced and
cluster-scoped, for Crossplane v2.

- `AuthorizationModel` takes the model in the OpenFGA DSL through `modelDsl`,
  in addition to the JSON the Terraform provider takes, and normalizes either
  one to the same canonical form.
- `Store` refuses to reconcile when its name changes, instead of deleting and
  recreating the store.
