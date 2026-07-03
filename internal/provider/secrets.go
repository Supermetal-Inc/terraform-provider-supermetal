package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
