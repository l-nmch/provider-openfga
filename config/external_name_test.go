package config

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	// keyStore is the Terraform argument every resource is scoped by, and
	// keyName the one a store is named by.
	keyStore = "store_id"
	keyName  = "name"

	// storeName is the name of the store under test, as opposed to the ULID
	// OpenFGA identifies it by.
	storeName = "phorge"

	storeID = "01K9B724V3EGG6B3R99R6642EQ"
	modelID = "01JF0PT3CRW2HAHBK4Z4RJBEWQ"
)

func TestStoreExternalName(t *testing.T) {
	e := ExternalNameConfigs["openfga_store"]

	// A store is identified by the ULID OpenFGA assigns it, which is both the
	// "id" attribute and the Terraform import ID.
	got, err := e.GetExternalNameFn(map[string]any{"id": storeID, keyName: storeName})
	if err != nil {
		t.Fatalf("GetExternalNameFn: %v", err)
	}
	if got != storeID {
		t.Errorf("external name: got %q, want %q", got, storeID)
	}

	id, err := e.GetIDFn(context.Background(), storeID, map[string]any{keyName: storeName}, nil)
	if err != nil {
		t.Fatalf("GetIDFn: %v", err)
	}
	if id != storeID {
		t.Errorf("import ID: got %q, want %q", id, storeID)
	}

	// Before creation there is no external name, and the ID has to be one
	// OpenFGA answers "not found" for rather than the empty string, which it
	// rejects outright. See computedIdentifier.
	id, err = e.GetIDFn(context.Background(), "", map[string]any{keyName: storeName}, nil)
	if err != nil {
		t.Fatalf("GetIDFn before creation: %v", err)
	}
	if id != placeholderID {
		t.Errorf("import ID before creation: got %q, want the placeholder %q", id, placeholderID)
	}

	// And the placeholder must never become an external name.
	if _, err := e.GetExternalNameFn(map[string]any{"id": placeholderID}); err == nil {
		t.Error("the placeholder was accepted as an external name")
	}
	if _, err := e.GetExternalNameFn(map[string]any{"id": ""}); err == nil {
		t.Error("an empty id was accepted as an external name")
	}

	// The name is a plain argument, never set from the external name: renaming
	// a store is not something OpenFGA can do.
	params := map[string]any{keyName: storeName}
	e.SetIdentifierArgumentFn(params, storeID)
	if diff := cmp.Diff(map[string]any{keyName: storeName}, params); diff != "" {
		t.Errorf("parameters changed: %s", diff)
	}
}

func TestAuthorizationModelExternalName(t *testing.T) {
	e := ExternalNameConfigs["openfga_authorization_model"]

	// The external name is the model ULID alone...
	got, err := e.GetExternalNameFn(map[string]any{"id": modelID, keyStore: storeID})
	if err != nil {
		t.Fatalf("GetExternalNameFn: %v", err)
	}
	if got != modelID {
		t.Errorf("external name: got %q, want %q", got, modelID)
	}

	// ...and so is the "id" attribute of the Terraform state.
	id, err := e.GetIDFn(context.Background(), modelID, map[string]any{keyStore: storeID}, nil)
	if err != nil {
		t.Fatalf("GetIDFn: %v", err)
	}
	if id != modelID {
		t.Errorf("state id: got %q, want %q", id, modelID)
	}

	// Only "terraform import" needs the store, in a separate hook.
	if e.GetImportIDFn == nil {
		t.Fatal("GetImportIDFn is not set")
	}
	imported, err := e.GetImportIDFn(context.Background(), modelID, map[string]any{keyStore: storeID}, nil)
	if err != nil {
		t.Fatalf("GetImportIDFn: %v", err)
	}
	if want := storeID + "/" + modelID; imported != want {
		t.Errorf("import ID: got %q, want %q", imported, want)
	}
}

func TestAuthorizationModelExternalNameErrors(t *testing.T) {
	e := ExternalNameConfigs["openfga_authorization_model"]

	// Without a store there is nothing to scope the model by.
	if _, err := e.GetImportIDFn(context.Background(), modelID, map[string]any{}, nil); err == nil {
		t.Error("expected an error without a store, got none")
	}

	// Before the model exists there is no external name, and the import ID is
	// built on every reconcile regardless, so it falls back to the placeholder
	// instead of failing the whole reconcile.
	got, err := e.GetImportIDFn(context.Background(), "", map[string]any{keyStore: storeID}, nil)
	if err != nil {
		t.Fatalf("GetImportIDFn before creation: %v", err)
	}
	if want := storeID + "/" + placeholderID; got != want {
		t.Errorf("import ID before creation: got %q, want %q", got, want)
	}
}

func TestRelationshipTupleExternalName(t *testing.T) {
	e := ExternalNameConfigs["openfga_relationship_tuple"]
	tuple := map[string]any{
		keyStore:   storeID,
		"user":     "user:leo",
		"relation": "admin",
		"object":   "project:minecraft",
	}
	want := storeID + "/user:leo/admin/project:minecraft"

	// The resource has no "id" attribute: the external name is composed from
	// the tuple itself, both out of the state and out of the parameters, and
	// the two must agree.
	got, err := e.GetExternalNameFn(tuple)
	if err != nil {
		t.Fatalf("GetExternalNameFn: %v", err)
	}
	if got != want {
		t.Errorf("external name: got %q, want %q", got, want)
	}

	id, err := e.GetIDFn(context.Background(), got, tuple, nil)
	if err != nil {
		t.Fatalf("GetIDFn: %v", err)
	}
	if id != want {
		t.Errorf("import ID: got %q, want %q", id, want)
	}

	// A tuple that pins an authorization model carries it in the key, in the
	// five-part form "terraform import" accepts. Importing with the four-part
	// form would leave authorization_model_id unset, which reads as a
	// replacement and would have the tuple rewritten on every reconcile.
	withModel := map[string]any{"authorization_model_id": modelID}
	for k, v := range tuple {
		withModel[k] = v
	}
	wantWithModel := storeID + "/" + modelID + "/user:leo/admin/project:minecraft"

	got, err = e.GetExternalNameFn(withModel)
	if err != nil {
		t.Fatalf("GetExternalNameFn with a model: %v", err)
	}
	if got != wantWithModel {
		t.Errorf("external name with a model: got %q, want %q", got, wantWithModel)
	}

	id, err = e.GetIDFn(context.Background(), got, withModel, nil)
	if err != nil {
		t.Fatalf("GetIDFn with a model: %v", err)
	}
	if id != wantWithModel {
		t.Errorf("import ID with a model: got %q, want %q", id, wantWithModel)
	}
}

func TestRelationshipTupleExternalNameErrors(t *testing.T) {
	e := ExternalNameConfigs["openfga_relationship_tuple"]
	complete := func() map[string]any {
		return map[string]any{
			keyStore:   storeID,
			"user":     "user:leo",
			"relation": "admin",
			"object":   "project:minecraft",
		}
	}

	for _, missing := range []string{keyStore, "user", "relation", "object"} {
		t.Run("Missing_"+missing, func(t *testing.T) {
			attrs := complete()
			delete(attrs, missing)
			if _, err := e.GetExternalNameFn(attrs); err == nil {
				t.Errorf("expected an error when %q is missing, got none", missing)
			}
		})
	}

	// A slash in any part would make the composite key ambiguous.
	t.Run("SlashInPart", func(t *testing.T) {
		attrs := complete()
		attrs["object"] = "project:a/b"
		if _, err := e.GetExternalNameFn(attrs); err == nil {
			t.Error("expected an error for a part containing a slash, got none")
		}
	})
}

func TestExternalNameConfigured(t *testing.T) {
	got := ExternalNameConfigured()
	if len(got) != len(ExternalNameConfigs) {
		t.Fatalf("got %d entries, want %d", len(got), len(ExternalNameConfigs))
	}
	for _, e := range got {
		if e[len(e)-1] != '$' {
			t.Errorf("%q is not anchored", e)
		}
	}
}
