package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateConnectorIDChange(
	plannedID types.String,
	stateID types.String,
	requiresReplace bool,
	attributePath path.Path,
	diagnostics *diag.Diagnostics,
) {
	if plannedID.IsNull() || plannedID.IsUnknown() {
		return
	}

	if !requiresReplace && !stateID.IsNull() && !stateID.IsUnknown() && plannedID.Equal(stateID) {
		return
	}

	addConnectorIDDiagnostics(plannedID, attributePath, diagnostics)
}

func addConnectorIDDiagnostics(value types.String, attributePath path.Path, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return
	}

	id := value.ValueString()
	if len(id) > 30 {
		diagnostics.AddAttributeError(
			attributePath,
			"Invalid connector ID",
			fmt.Sprintf("Connector ID must be at most 30 characters; got %d.", len(id)),
		)
		return
	}
	if !connectorIDPattern.MatchString(id) {
		diagnostics.AddAttributeError(
			attributePath,
			"Invalid connector ID",
			"Connector ID must contain only lowercase letters, numbers, and underscores.",
		)
	}
}
