# v0.1.2

- Build with Go 1.26.10 and golang.org/x/net v0.60.0, which fix the Go
  standard library and HTTP/2 vulnerabilities reported by govulncheck.
- Raise the vulnerable dependencies of the terraform-provider-openfga binary
  (golang.org/x/net, golang.org/x/text, google.golang.org/grpc,
  go.opentelemetry.io/otel) when the image builds it.
- Update Upjet to v2.5.1, which fixes null and empty collections being treated
  as drift, and crossplane-runtime to the stable v2.4.3.

# v0.1.1

- Fix `AuthorizationModel` never creating the first model of a store: the
  image now builds terraform-provider-openfga from source with the patches of
  `hack/upstream` applied, instead of using the released binary.

# v0.1.0

First release. All three resources of terraform-provider-openfga v0.5.1
(`Store`, `AuthorizationModel`, `RelationshipTuple`), namespaced and
cluster-scoped, for Crossplane v2.

- `AuthorizationModel` takes the model in the OpenFGA DSL through `modelDsl`,
  in addition to the JSON the Terraform provider takes, and normalizes either
  one to the same canonical form.
- `Store` refuses to reconcile when its name changes, instead of deleting and
  recreating the store.
