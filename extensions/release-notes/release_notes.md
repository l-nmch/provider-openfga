# Changelog

## [0.1.3](https://github.com/l-nmch/provider-openfga/compare/v0.1.2...v0.1.3) (2026-10-29)


### Bug Fixes

* **deps:** update all non-major go dependencies ([b73a253](https://github.com/l-nmch/provider-openfga/commit/b73a253ebd2b9c0c872c16eac7f2b2f0dd236091))
* **deps:** update all non-major go dependencies ([e702c50](https://github.com/l-nmch/provider-openfga/commit/e702c503f08f23c5f6c4100a52476726f864c960))
* **deps:** update kubernetes monorepo to v0.37.2 ([5a0c97c](https://github.com/l-nmch/provider-openfga/commit/5a0c97cb3bd8479ae5284abf98123739d92a1c44))
* **deps:** update kubernetes monorepo to v0.37.2 ([88fb891](https://github.com/l-nmch/provider-openfga/commit/88fb891b5c49b423b6e941dbf84398f712aaa201))

## 0.1.2

- Build with Go 1.26.10 and golang.org/x/net v0.60.0, which fix the Go
  standard library and HTTP/2 vulnerabilities reported by govulncheck.
- Raise the vulnerable dependencies of the terraform-provider-openfga binary
  (golang.org/x/net, golang.org/x/text, google.golang.org/grpc,
  go.opentelemetry.io/otel) when the image builds it.
- Update Upjet to v2.5.1, which fixes null and empty collections being treated
  as drift, and crossplane-runtime to the stable v2.4.3.

## 0.1.1

- Fix `AuthorizationModel` never creating the first model of a store: the
  image now builds terraform-provider-openfga from source with the patches of
  `hack/upstream` applied, instead of using the released binary.

## 0.1.0

First release. All three resources of terraform-provider-openfga v0.5.1
(`Store`, `AuthorizationModel`, `RelationshipTuple`), namespaced and
cluster-scoped, for Crossplane v2.

- `AuthorizationModel` takes the model in the OpenFGA DSL through `modelDsl`,
  in addition to the JSON the Terraform provider takes, and normalizes either
  one to the same canonical form.
- `Store` refuses to reconcile when its name changes, instead of deleting and
  recreating the store.
