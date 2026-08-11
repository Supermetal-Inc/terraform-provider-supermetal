package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

const redactionMarker = "***"

func isRedacted(s string) bool {
	return strings.Contains(s, redactionMarker)
}

// The server's clear_secrets.rs scrub can mangle strings that are not
// secrets but contain UUIDs, hex tokens, or similar patterns.
func mergeString(apiVal string, stateVal types.String) types.String {
	if isRedacted(apiVal) {
		return stateVal
	}
	return types.StringValue(apiVal)
}

func mergeStringPtr(apiVal *string, stateVal types.String) types.String {
	if apiVal == nil {
		return types.StringNull()
	}
	return mergeString(*apiVal, stateVal)
}

// isObjectStoreSecretOption identifies option names removed by the agent's
// object store response scrubber. Preserve only these absent entries because
// any other absent option represents a deletion.
func isObjectStoreSecretOption(name string) bool {
	switch name {
	case "access_token",
		"refresh_token",
		"client_secret",
		"private_key",
		"access_key_id",
		"aws_access_key_id",
		"secret_access_key",
		"aws_secret_access_key",
		"token",
		"session_token",
		"aws_token",
		"aws_session_token",
		"container_authorization_token_file",
		"aws_container_authorization_token_file",
		"web_identity_token_file",
		"aws_web_identity_token_file",
		"aws_sse_customer_key_base64",
		"sse_customer_key_base64",
		"access_key",
		"account_key",
		"master_key",
		"azure_storage_account_key",
		"azure_storage_access_key",
		"azure_storage_master_key",
		"azure_storage_client_secret",
		"azure_client_secret",
		"sas_key",
		"sas_token",
		"azure_storage_sas_key",
		"azure_storage_sas_token",
		"bearer_token",
		"azure_storage_token",
		"federated_token_file",
		"azure_federated_token_file",
		"service_account",
		"service_account_path",
		"google_service_account",
		"google_service_account_path",
		"service_account_key",
		"google_service_account_key",
		"application_credentials",
		"google_application_credentials":
		return true
	default:
		return false
	}
}

// connectorJSONWithSecretTombstones preserves replacement semantics after the
// agent restores secrets omitted from API requests. Terraform sends null when
// configuration removes an optional secret but retains its parent object. The
// generated OpenAPI paths limit this behavior to declared secrets.
func connectorJSONWithSecretTombstones(plan, state api.ConnectorConnector) ([]byte, error) {
	planJSON, err := connectorJSONObject(plan)
	if err != nil {
		return nil, fmt.Errorf("marshal planned connector: %w", err)
	}
	stateJSON, err := connectorJSONObject(state)
	if err != nil {
		return nil, fmt.Errorf("marshal prior connector state: %w", err)
	}

	addDeclaredSecretTombstones(planJSON, stateJSON)
	body, err := json.Marshal(planJSON)
	if err != nil {
		return nil, fmt.Errorf("marshal connector with secret removals: %w", err)
	}
	return body, nil
}

func connectorJSONObject(connector api.ConnectorConnector) (map[string]any, error) {
	body, err := json.Marshal(connector)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	return object, nil
}

func addDeclaredSecretTombstones(plan, state map[string]any) {
	for _, path := range declaredSecretJSONPaths {
		if len(path) == 0 {
			continue
		}
		stateParent, ok := jsonObjectAtPath(state, path[:len(path)-1])
		if !ok {
			continue
		}
		leaf := path[len(path)-1]
		stateValue, configured := stateParent[leaf]
		if !configured || stateValue == nil {
			continue
		}

		// An absent parent object already expresses removal. A retained object
		// needs a null leaf to distinguish removal from redaction.
		planParent, ok := jsonObjectAtPath(plan, path[:len(path)-1])
		if !ok {
			continue
		}
		if _, present := planParent[leaf]; !present {
			planParent[leaf] = nil
		}
	}
}

func jsonObjectAtPath(root map[string]any, path []string) (map[string]any, bool) {
	current := root
	for _, segment := range path {
		next, ok := current[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}
