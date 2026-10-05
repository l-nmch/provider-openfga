# provider-openfga

`provider-openfga` is a [Crossplane](https://crossplane.io/) provider for
[OpenFGA](https://openfga.dev/), generated with
[Upjet](https://github.com/crossplane/upjet) from
[`openfga/terraform-provider-openfga`](https://github.com/openfga/terraform-provider-openfga)
**v0.5.1**. It targets Crossplane v2 and exposes every resource of the
Terraform provider, both as namespaced (`*.openfga.m.crossplane.io`) and
cluster-scoped (`*.openfga.crossplane.io`) managed resources.

## Resources

| Group | Kind | Terraform resource | Identified by |
|---|---|---|---|
| `store` | `Store` | `openfga_store` | store ULID, assigned by OpenFGA |
| `model` | `AuthorizationModel` | `openfga_authorization_model` | model ULID, assigned by OpenFGA |
| `tuple` | `RelationshipTuple` | `openfga_relationship_tuple` | `<store_id>[/<model_id>]/<user>/<relation>/<object>` |

OpenFGA assigns a ULID to stores and authorization models, so `metadata.name`
is not their OpenFGA identity: a store's name is `spec.forProvider.name`, and
the ULID becomes the `crossplane.io/external-name` annotation once the object
exists. Set that annotation to an existing ULID to adopt it.

A relationship tuple has no identifier of its own. Its external name is the
composite key above, which is also what `terraform import` takes. The
authorization model is part of it only when the tuple pins one: importing a
pinned tuple without it leaves `authorizationModelId` unset, which reads as a
change to an immutable field and would have the tuple rewritten on every
reconcile.

Cross-resource references: `storeIdRef`/`storeIdSelector` on
`AuthorizationModel` and `RelationshipTuple`, and
`authorizationModelIdRef`/`authorizationModelIdSelector` on
`RelationshipTuple`. Both resolve to the referenced object's ULID once it
exists, so a model is created after its store and a tuple after both.

## Configuration

Create a Secret holding a JSON document, and a `ProviderConfig` (or
`ClusterProviderConfig`) pointing to it:

```json
{
  "api_url": "https://openfga.example.org",
  "api_token": "..."
}
```

| Key | Description |
|---|---|
| `api_url` | OpenFGA API URL (required). |
| `api_token` | Pre-shared key, or any access token obtained out of band. |
| `client_id`, `client_secret`, `api_token_issuer` | OIDC client credentials; the provider obtains an access token itself. All three are needed. |
| `api_audience`, `api_scopes` | Refine that token request. |
| `ca_cert` | PEM certificate authority to trust in addition to the system ones. |

`api_token` and the OIDC client credentials are mutually exclusive, and a
server configured without authentication needs nothing but `api_url`. Keys
this provider does not know are rejected rather than ignored, so a misspelled
`api_token` fails instead of looking like an unauthenticated server.

```yaml
apiVersion: openfga.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
  namespace: my-namespace
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: my-namespace
      name: openfga-creds
      key: credentials
```

`ca_cert` is written into a trust bundle the Terraform provider plugin reads
through `SSL_CERT_FILE`. The plugin is a separate process whose trust store
belongs to the provider pod rather than to one ProviderConfig, so the bundle
holds the system certificates plus every authority any ProviderConfig
supplied: several of them compose instead of overwriting each other. Mounting
the authority into the pod and setting `SSL_CERT_FILE` through a
`DeploymentRuntimeConfig` has the same effect and keeps it out of the Secret.
The Terraform provider has no option to skip TLS verification, and neither
does this one.

## Known limitation: AuthorizationModel cannot be created yet

`Store` and `RelationshipTuple` work. **`AuthorizationModel` does not, with
terraform-provider-openfga 0.5.1**, because of a gap upstream rather than
anything in this provider.

Upjet observes a managed resource by refreshing a Terraform state it builds
out of the resource's parameters, before the resource exists. `openfga_store`
and `openfga_relationship_tuple` answer that refresh by removing themselves
from the state, which is how Crossplane learns it has to create them.
`openfga_authorization_model` reports an error instead, so its reconcile never
gets as far as creating anything:

```
observe failed: cannot run refresh: refresh failed: Client Error: Unable to
read authorization model, got error: ... {"code":"authorization_model_not_found"}
```

Upstream's `apierror.IsStatusNotFound` already recognises that error code; the
model resource simply never calls it. The three-line fix is in
[`hack/upstream/`](hack/upstream/), ready to send as a pull request. Once a
release carries it, bump `TERRAFORM_PROVIDER_VERSION` in the Makefile and run
`make generate` — this provider needs no other change.

Until then, create authorization models with the `fga` CLI or the OpenFGA API
and let tuples resolve against the store's latest model (leave
`authorizationModelId` unset). The `AuthorizationModel` kind is still shipped,
so it starts working on a version bump alone.

## Usage notes

- **Everything is immutable.** OpenFGA has no update operation for stores,
  authorization models or relationship tuples, and the Terraform provider
  marks every argument as requiring replacement. Changing a field therefore
  deletes and recreates the object. That is harmless for a tuple, invisible
  for a model (see below), and destructive for a store, so a `Store` whose
  `spec.forProvider.name` no longer matches the name it was created with is
  reported as an error instead of being recreated. To rename a store, create
  a new one.
- **Authorization models are versioned and append-only.** Changing `modelDsl`
  or `modelJson` writes a new version of the model and leaves the previous one
  in the store; the resource's external name then points at the new version.
  Deleting the managed resource succeeds but does not remove the model from
  the store either, since OpenFGA cannot delete authorization models. A store
  therefore accumulates model versions, which is how OpenFGA is meant to be
  used, not a leak.
- **`modelDsl` is the readable way to write a model.** The Terraform provider
  only accepts JSON in `model_json` and converts the DSL in a *data source*,
  which upjet does not generate. This provider adds `modelDsl` and performs
  that conversion itself, with the same `openfga/language` library. Set
  exactly one of `modelDsl` and `modelJson`; both are normalized to the same
  canonical JSON, so the same model written either way produces no diff.
- A `RelationshipTuple` without `authorizationModelId` is validated against
  the store's latest model. Setting it both pins the validation and orders the
  tuple after the model.
- `status.atProvider.modelJson` shows the canonical form of the model that was
  actually applied. It is not late-initialized into the spec, so what you
  wrote stays what you wrote.

## Developing

Tools: Go (see `go.mod`), `goimports` on `PATH`, Docker for images, `python3`
for the package metadata.

```console
make generate        # schema, docs, CRDs, controllers, package metadata
make reviewable      # generate + lint + tests
make build           # provider image and xpkg under _output/
```

Provider configuration lives in [`config/`](config/): external names in
[`external_name.go`](config/external_name.go) and one package per API group
([`store`](config/store), [`model`](config/model), [`tuple`](config/tuple)).
Credentials handling is in
[`internal/clients/openfga.go`](internal/clients/openfga.go).

### Implementation notes

The Terraform provider is built on terraform-plugin-framework and exposes
nothing outside its `internal/` packages, so it cannot be linked in as a Go
module and upjet's native framework client is unavailable. This provider runs
Terraform the way `upjet-provider-template` does: the `terraform` CLI plus the
released provider plugin, served from a filesystem mirror inside the image
(see [`cluster/images/provider-openfga`](cluster/images/provider-openfga)).
Terraform 1.5.7 introspects the framework provider's schema without trouble.

Two consequences of that shape the configuration in `config/`:

- `openfga_relationship_tuple` has no `id` attribute at all. Upjet handles
  ID-less resources natively, but the external name has to be composed from
  the tuple's own fields, which `config/external_name.go` does in both
  directions.
- `openfga_authorization_model`'s `id` attribute holds only the model ULID,
  while `terraform import` expects `<store_id>/<model_id>`. The external name
  is the former and `GetIDFn` builds the latter.

`modelJson` is excluded from late-initialization, and `config/model` renders
the model into the exact shape the Terraform provider puts in the state when
it reads a model back, which is protojson's output *with* default values —
`"conditions": {}`, `"module": ""` and so on — because OpenFGA fills those in.
The minimal form a human would write reads as a difference instead, and since
every argument requires replacement, and upjet reads a replacement plan as a
resource that does not exist, each reconcile would write another version of
the model. This was settled by running a real Terraform apply against an
OpenFGA server and checking the second plan came out empty; the unit tests in
[`config/model`](config/model) pin the shape so it stays that way.

### End-to-end tests

The manifests in [`examples/e2e/`](examples/e2e/) cover all three resources.
Everything they create is named or prefixed `xptest`, in an `xptest`
Kubernetes namespace and an `xptest` store.

1. `FGA_API_URL=... FGA_API_TOKEN=... hack/e2e/snapshot.sh > before.json`
   records the OpenFGA state.
2. Apply `package/crds`, the credentials Secret and the ProviderConfig, and
   run the provider out of cluster with `--certs-dir ""`. Set
   `PLUGIN_UNIX_SOCKET_DIR` to a short directory when `TMPDIR` is long:
   Terraform plugins listen on a UNIX socket whose path is limited to 108
   characters. Use a provider filesystem mirror (`TF_CLI_CONFIG_FILE`, as in
   the image) rather than a shared plugin cache, which concurrent
   `terraform init` runs corrupt.
3. Apply the manifests in order, check `Synced`/`Ready`, then delete them.
4. `hack/e2e/cleanup.sh [kube-context]` idempotently removes whatever is left
   on both sides, and `hack/e2e/snapshot.sh | diff before.json -` must come
   out empty.

Worth checking by hand, since neither is covered by a unit test: editing a
`Store`'s name must leave the store intact and report an error, and editing an
`AuthorizationModel` must add a model version rather than reconcile forever.

## License

This provider is licensed under the [Apache License 2.0](LICENSE).

Its container image also ships third-party binaries, redistributed unmodified
under their own license, whose source code is available upstream:

| Component | Version | License | Source |
|---|---|---|---|
| Terraform CLI | 1.5.7 | MPL-2.0 | https://github.com/hashicorp/terraform/tree/v1.5.7 |
| terraform-provider-openfga | 0.5.1 | Apache-2.0 | https://github.com/openfga/terraform-provider-openfga/tree/v0.5.1 |

Resource and field descriptions in the CRDs are derived from the
terraform-provider-openfga documentation (Apache-2.0).
