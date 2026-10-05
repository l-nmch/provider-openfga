#!/usr/bin/env bash
# Dumps a normalized snapshot of the OpenFGA server, so that a before/after
# diff proves an end-to-end run left nothing behind.
#
# usage: FGA_API_URL=http://10.0.0.12:8080 FGA_API_TOKEN=... hack/e2e/snapshot.sh > before.json
#
# Authorization model IDs and tuple timestamps are dropped: they are assigned
# per write, so they differ between two runs that are otherwise identical.
set -euo pipefail

: "${FGA_API_URL:?FGA_API_URL is required}"
AUTH=()
if [[ -n "${FGA_API_TOKEN:-}" ]]; then
  AUTH=(-H "Authorization: Bearer ${FGA_API_TOKEN}")
fi

api() {
  curl -fsS "${AUTH[@]}" -H 'Content-Type: application/json' "$@"
}

stores=$(api "${FGA_API_URL}/stores?page_size=100")

jq_prog='
  def clean_model: {schema_version, type_definitions, conditions};
  def clean_tuple: .key | {user, relation, object, condition};
  {
    stores: [ .[] | {
      name: .store.name,
      models: [ .models[] | clean_model ],
      tuples: ( [ .tuples[] | clean_tuple ] | sort_by(.object, .relation, .user) )
    } ] | sort_by(.name)
  }
'

out='[]'
while read -r id; do
  [[ -z "$id" ]] && continue
  store=$(api "${FGA_API_URL}/stores/${id}")
  models=$(api "${FGA_API_URL}/stores/${id}/authorization-models?page_size=100" | jq '.authorization_models // []')
  tuples=$(api -X POST -d '{"page_size":100}' "${FGA_API_URL}/stores/${id}/read" | jq '.tuples // []')
  out=$(jq -n --argjson acc "$out" --argjson store "$store" --argjson models "$models" --argjson tuples "$tuples" \
    '$acc + [{store: $store, models: $models, tuples: $tuples}]')
done < <(echo "$stores" | jq -r '.stores[]?.id')

echo "$out" | jq -S "$jq_prog"
