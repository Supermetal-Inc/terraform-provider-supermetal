package provider

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

type objectStoreCredentialLocation struct {
	name  string
	path  []string
	store func(*ConnectorModel) *BufferObjectStoreModel
}

var objectStoreCredentialLocations = []objectStoreCredentialLocation{
	{
		name: "buffer",
		path: []string{"buffer", "object_store"},
		store: func(model *ConnectorModel) *BufferObjectStoreModel {
			return model.Buffer.ObjectStore
		},
	},
	{
		name: "file source",
		path: []string{"source", "file_source", "object_store"},
		store: func(model *ConnectorModel) *BufferObjectStoreModel {
			return model.Source.FileSource.ObjectStore
		},
	},
	{
		name: "file sink",
		path: []string{"sink", "file_sink", "object_store"},
		store: func(model *ConnectorModel) *BufferObjectStoreModel {
			return model.Sink.FileSink.ObjectStore
		},
	},
	{
		name: "Iceberg SlateDB",
		path: []string{"sink", "iceberg", "write_mode", "merge_on_read", "index_backend", "slatedb", "object_store"},
		store: func(model *ConnectorModel) *BufferObjectStoreModel {
			return model.Sink.Iceberg.WriteMode.MergeOnRead.IndexBackend.Slatedb.ObjectStore
		},
	},
}

var gcsServiceAccountAliases = []string{
	"service_account_key",
	"google_service_account_key",
	"service_account",
	"google_service_account",
	"service_account_path",
	"google_service_account_path",
}

func TestDeclaredSecretJSONPathsIncludesBigQueryKey(t *testing.T) {
	const want = "sink.big_query.auth.service_account_key.key_json"
	for _, path := range declaredSecretJSONPaths {
		if strings.Join(path, ".") == want {
			return
		}
	}
	t.Fatalf("declared secret paths do not contain %q", want)
}

func TestObjectStoreJSONPathsIncludeCredentialLocations(t *testing.T) {
	want := map[string]bool{
		"buffer.object_store":             false,
		"source.file_source.object_store": false,
		"sink.file_sink.object_store":     false,
		"sink.iceberg.write_mode.merge_on_read.index_backend.slatedb.object_store": false,
	}
	for _, path := range objectStoreJSONPaths {
		joined := strings.Join(path, ".")
		if _, required := want[joined]; required {
			want[joined] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("object store paths do not contain %q", path)
		}
	}
}

func TestObjectStoreSecretOptionClassification(t *testing.T) {
	for _, name := range gcsServiceAccountAliases {
		if isObjectStoreSecretOption(name) {
			t.Errorf("GCS service account alias %q is still preserved by absence", name)
		}
		if !isGCSServiceAccountOption(name) {
			t.Errorf("GCS service account alias %q is not classified for explicit removal", name)
		}
	}
	if !isObjectStoreSecretOption("google_application_credentials") {
		t.Error("google_application_credentials must remain preserved by absence")
	}
}

func TestAddDeclaredSecretTombstones(t *testing.T) {
	t.Run("removed secret leaf emits null", func(t *testing.T) {
		plan := map[string]any{
			"buffer": map[string]any{
				"object_store": map[string]any{"url": "s3://bucket"},
			},
		}
		state := map[string]any{
			"buffer": map[string]any{
				"object_store": map[string]any{
					"url":                  "s3://bucket",
					"root_certificate_pem": "certificate",
				},
			},
		}

		addDeclaredSecretTombstones(plan, state)

		objectStore := plan["buffer"].(map[string]any)["object_store"].(map[string]any)
		value, present := objectStore["root_certificate_pem"]
		if !present || value != nil {
			t.Fatalf("root_certificate_pem = %#v, present = %t; want explicit null", value, present)
		}
	})

	t.Run("removed parent remains absent", func(t *testing.T) {
		plan := map[string]any{}
		state := map[string]any{
			"buffer": map[string]any{
				"object_store": map[string]any{"root_certificate_pem": "certificate"},
			},
		}

		addDeclaredSecretTombstones(plan, state)

		if _, present := plan["buffer"]; present {
			t.Fatal("removed buffer block was recreated")
		}
	})

	t.Run("ordinary removed field remains absent", func(t *testing.T) {
		plan := map[string]any{
			"buffer": map[string]any{
				"object_store": map[string]any{"url": "s3://bucket"},
			},
		}
		state := map[string]any{
			"buffer": map[string]any{
				"object_store": map[string]any{
					"url":                  "s3://bucket",
					"max_concurrent_parts": float64(4),
				},
			},
		}

		addDeclaredSecretTombstones(plan, state)

		objectStore := plan["buffer"].(map[string]any)["object_store"].(map[string]any)
		if _, present := objectStore["max_concurrent_parts"]; present {
			t.Fatal("ordinary removed field received a tombstone")
		}
	})
}

func TestAddGCSServiceAccountOptionRemovals(t *testing.T) {
	for _, location := range objectStoreCredentialLocations {
		for _, alias := range gcsServiceAccountAliases {
			t.Run(location.name+"/remove/"+alias, func(t *testing.T) {
				plan := connectorWithObjectStore(t, location.path, map[string]any{"url": "gs://bucket"})
				state := connectorWithObjectStore(t, location.path, map[string]any{
					"url":     "gs://bucket",
					"options": []any{map[string]any{"name": alias, "value": "secret"}},
				})

				body, err := connectorJSONWithSecretTombstones(plan, state)
				if err != nil {
					t.Fatalf("connector JSON: %v", err)
				}
				var object map[string]any
				if err := json.Unmarshal(body, &object); err != nil {
					t.Fatalf("unmarshal connector JSON: %v", err)
				}

				store, ok := jsonObjectAtPath(object, location.path)
				if !ok {
					t.Fatal("object store disappeared")
				}
				if got, want := store["options"], []any{}; !reflect.DeepEqual(got, want) {
					t.Fatalf("options = %#v, want %#v", got, want)
				}
			})
		}
	}
}

func TestObjectStoreGCSCredentialRefreshAndImport(t *testing.T) {
	for _, location := range objectStoreCredentialLocations {
		for _, alias := range gcsServiceAccountAliases {
			t.Run(location.name+"/refresh/"+alias, func(t *testing.T) {
				stateAPI := connectorWithObjectStore(t, location.path, map[string]any{
					"url":     "gs://bucket",
					"options": []any{map[string]any{"name": alias, "value": "real-secret"}},
				})
				state := &ConnectorModel{}
				state.mergeFromAPI(stateAPI, nil)

				response := connectorWithObjectStore(t, location.path, map[string]any{
					"url":     "gs://bucket",
					"options": []any{map[string]any{"name": alias, "value": redactionMarker}},
				})
				refreshed := &ConnectorModel{}
				refreshed.mergeFromAPI(response, state)

				option := location.store(refreshed).Options[alias]
				if option == nil || option.Value.ValueString() != "real-secret" {
					t.Fatalf("refreshed %q = %#v, want preserved real value", alias, option)
				}
			})

			t.Run(location.name+"/import/"+alias, func(t *testing.T) {
				response := connectorWithObjectStore(t, location.path, map[string]any{
					"url":     "gs://bucket",
					"options": []any{map[string]any{"name": alias, "value": redactionMarker}},
				})
				imported := &ConnectorModel{}
				imported.mergeFromAPI(response, nil)

				option := location.store(imported).Options[alias]
				if option == nil || !option.Value.IsNull() {
					t.Fatalf("imported %q = %#v, want null value awaiting configuration", alias, option)
				}
			})
		}

		t.Run(location.name+"/ADC response drops key but keeps hidden credentials", func(t *testing.T) {
			stateAPI := connectorWithObjectStore(t, location.path, map[string]any{
				"url": "gs://bucket",
				"options": []any{
					map[string]any{"name": "service_account_key", "value": "real-secret"},
					map[string]any{"name": "google_application_credentials", "value": "/hidden/path"},
				},
			})
			state := &ConnectorModel{}
			state.mergeFromAPI(stateAPI, nil)

			response := connectorWithObjectStore(t, location.path, map[string]any{"url": "gs://bucket"})
			refreshed := &ConnectorModel{}
			refreshed.mergeFromAPI(response, state)

			options := location.store(refreshed).Options
			if _, present := options["service_account_key"]; present {
				t.Error("absent GCS service account key survived an ADC response")
			}
			credential := options["google_application_credentials"]
			if credential == nil || credential.Value.ValueString() != "/hidden/path" {
				t.Fatalf("hidden google_application_credentials = %#v, want preserved value", credential)
			}
		})
	}
}

func connectorWithObjectStore(t *testing.T, path []string, store map[string]any) api.ConnectorConnector {
	t.Helper()
	object := objectAtPath(path, store)
	object["id"] = "test_connector"
	body, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal connector fixture: %v", err)
	}
	var connector api.ConnectorConnector
	if err := json.Unmarshal(body, &connector); err != nil {
		t.Fatalf("unmarshal connector fixture: %v", err)
	}
	return connector
}

func objectAtPath(path []string, leaf map[string]any) map[string]any {
	root := map[string]any{}
	current := root
	for i, segment := range path {
		if i == len(path)-1 {
			current[segment] = leaf
			break
		}
		next := map[string]any{}
		current[segment] = next
		current = next
	}
	return root
}
