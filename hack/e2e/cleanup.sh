#!/usr/bin/env bash
# Idempotently removes whatever an end-to-end run may have left behind, on both
# sides. Safe to run repeatedly, and when nothing is left.
#
# usage: hack/e2e/cleanup.sh [kube-context]
#        FGA_API_URL=http://10.0.0.12:8080 FGA_API_TOKEN=... hack/e2e/cleanup.sh control
#
# Everything the suite creates is named or prefixed "xptest", in the "xptest"
# namespace, and that is all this removes.
set -euo pipefail

CONTEXT="${1:-control}"
NAMESPACE="${XPTEST_NAMESPACE:-xptest}"
KUBECTL=${KUBECTL:-kubectl}
K="${KUBECTL} --context=${CONTEXT}"

echo "==> Deleting managed resources in ${NAMESPACE}"
for kind in relationshiptuples.tuple.openfga.m.crossplane.io \
            authorizationmodels.model.openfga.m.crossplane.io \
            stores.store.openfga.m.crossplane.io; do
  if ${K} get crd "${kind}" >/dev/null 2>&1; then
    ${K} -n "${NAMESPACE}" delete "${kind}" --all --timeout=60s --ignore-not-found || true
    # A provider that can no longer reach OpenFGA would block on finalizers.
    for o in $(${K} -n "${NAMESPACE}" get "${kind}" -o name 2>/dev/null); do
      ${K} -n "${NAMESPACE}" patch "${o}" -p '{"metadata":{"finalizers":[]}}' --type=merge || true
    done
  fi
done
${K} delete namespace "${NAMESPACE}" --ignore-not-found --timeout=60s || true

if [[ -z "${FGA_API_URL:-}" ]]; then
  echo "==> FGA_API_URL is not set, leaving the OpenFGA side alone"
  exit 0
fi

AUTH=()
if [[ -n "${FGA_API_TOKEN:-}" ]]; then
  AUTH=(-H "Authorization: Bearer ${FGA_API_TOKEN}")
fi

echo "==> Deleting xptest stores on ${FGA_API_URL}"
# Deleting a store takes its authorization models and relationship tuples with
# it, so there is nothing else to clean up.
curl -fsS "${AUTH[@]}" "${FGA_API_URL}/stores?page_size=100" \
  | jq -r '.stores[]? | select(.name | startswith("xptest")) | .id' \
  | while read -r id; do
      [[ -z "$id" ]] && continue
      echo "    store ${id}"
      curl -fsS "${AUTH[@]}" -X DELETE "${FGA_API_URL}/stores/${id}" || true
    done

echo "==> Done"
