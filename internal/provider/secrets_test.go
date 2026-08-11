package provider

import (
	"strings"
	"testing"
)

func TestDeclaredSecretJSONPathsIncludesBigQueryKey(t *testing.T) {
	const want = "sink.big_query.auth.service_account_key.key_json"
	for _, path := range declaredSecretJSONPaths {
		if strings.Join(path, ".") == want {
			return
		}
	}
	t.Fatalf("declared secret paths do not contain %q", want)
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
