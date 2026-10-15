# Upstream patches

Changes this provider needs in
[`openfga/terraform-provider-openfga`](https://github.com/openfga/terraform-provider-openfga),
kept here until they are merged and released. The provider image builds
`terraform-provider-openfga` from source at `TERRAFORM_PROVIDER_VERSION` (see
`cluster/images/provider-openfga/Dockerfile`) and applies every `*.patch` of
this directory before compiling, in file name order. Patches are plain `git
apply` diffs against that tag; a patch that no longer applies fails the image
build. Remove a patch once the pinned upstream release carries it.

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

Status: present in v0.5.1 and on `main` as of 2026-10-05, applied at image
build since this provider's next release. Once an upstream release carries
the fix, bump `TERRAFORM_PROVIDER_VERSION`, run `make generate` and delete the
patch.

Checked against a real OpenFGA server with a hand-written state holding a model
that does not exist: v0.5.1 fails the refresh with `authorization_model_not_found`,
the patched build drops the resource from the state, which is what lets Upjet
plan a create.
