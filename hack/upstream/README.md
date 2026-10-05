# Upstream patches

Changes this provider needs in
[`openfga/terraform-provider-openfga`](https://github.com/openfga/terraform-provider-openfga),
kept here until they are merged and released. Nothing here is applied
automatically: the provider image uses the released upstream binary, as
`TERRAFORM_PROVIDER_VERSION` in the Makefile pins it.

## 0001-authorization-model-read-not-found.patch

`openfga_authorization_model`'s `Read` reports an error when the model does
not exist, instead of removing it from the Terraform state the way
`openfga_store` and `openfga_relationship_tuple` do. `apierror.IsStatusNotFound`
already recognises the `authorization_model_not_found` code, so the resource
just never calls it.

Upjet observes a resource by refreshing a Terraform state it builds from the
resource's parameters, before the resource exists. For the store and the tuple
that refresh answers "gone" and Crossplane creates them; for an authorization
model it fails, so `AuthorizationModel` cannot be created. See the
corresponding note in the repository README.

Status: present in v0.5.1 and on `main` as of 2026-10-05. Once a release
carries the fix, bump `TERRAFORM_PROVIDER_VERSION` and run `make generate`;
this provider needs no other change.
