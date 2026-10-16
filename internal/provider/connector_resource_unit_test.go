package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

func TestConnectorIDValidation(t *testing.T) {
	tests := map[string]struct {
		plan            types.String
		state           types.String
		requiresReplace bool
		expectError     bool
	}{
		"new lowercase letters numbers and underscores": {
			plan:  types.StringValue("connector_123"),
			state: types.StringNull(),
		},
		"new 30 character ID": {
			plan:  types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			state: types.StringNull(),
		},
		"new uppercase ID": {
			plan:        types.StringValue("Connector_123"),
			state:       types.StringNull(),
			expectError: true,
		},
		"new hyphenated ID": {
			plan:        types.StringValue("connector-123"),
			state:       types.StringNull(),
			expectError: true,
		},
		"new empty ID": {
			plan:        types.StringValue(""),
			state:       types.StringNull(),
			expectError: true,
		},
		"new 31 character ID": {
			plan:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			state:       types.StringNull(),
			expectError: true,
		},
		"unchanged legacy hyphenated ID": {
			plan:  types.StringValue("legacy-connector"),
			state: types.StringValue("legacy-connector"),
		},
		"unchanged legacy uppercase ID": {
			plan:  types.StringValue("LegacyConnector"),
			state: types.StringValue("LegacyConnector"),
		},
		"unchanged legacy long ID": {
			plan:  types.StringValue("legacy_connector_identifier_over_thirty_characters"),
			state: types.StringValue("legacy_connector_identifier_over_thirty_characters"),
		},
		"invalid replacement ID": {
			plan:        types.StringValue("another-legacy-id"),
			state:       types.StringValue("legacy-connector"),
			expectError: true,
		},
		"replacement cannot reuse unchanged legacy ID": {
			plan:            types.StringValue("legacy-connector"),
			state:           types.StringValue("legacy-connector"),
			requiresReplace: true,
			expectError:     true,
		},
		"replace legacy ID with valid ID": {
			plan:  types.StringValue("valid_connector"),
			state: types.StringValue("legacy-connector"),
		},
		"destroy": {
			plan:  types.StringNull(),
			state: types.StringValue("legacy-connector"),
		},
		"unknown planned ID": {
			plan:  types.StringUnknown(),
			state: types.StringNull(),
		},
		"unknown state does not grandfather invalid ID": {
			plan:        types.StringValue("new-invalid-id"),
			state:       types.StringUnknown(),
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			validateConnectorIDChange(test.plan, test.state, test.requiresReplace, path.Root("id"), &diagnostics)

			if diagnostics.HasError() != test.expectError {
				t.Fatalf("validation diagnostics = %v, expect error = %t", diagnostics, test.expectError)
			}
		})
	}
}

func TestConnectorResourceValidateConnectorReportsWarningsAndCallsBuffer(t *testing.T) {
	calls := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/validate/source" {
			_, _ = w.Write([]byte(`[{"test":{"id":"source-config","status":{"Warn":{}},"test_type":{"Config":{"name":"Table Replica Identity","description":"Some tables require REPLICA IDENTITY FULL.","suggestions":["ALTER TABLE public.needs_full REPLICA IDENTITY FULL;"]}},"timestamp":"now"}}]`))
			return
		}
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatalf("create API client: %v", err)
	}
	model := buildFullConnectorEnvelopeModel()
	model.Source = &SourceModel{Postgres: buildFullPostgresSourceModel_WithSnapshot()}
	model.Sink = &SinkModel{Duckdb: buildFullDuckDBSinkModel_WithQuack()}
	connector, conversionDiags := model.toAPIConnector()
	if conversionDiags.HasError() {
		t.Fatalf("convert connector: %v", conversionDiags)
	}

	resource := &ConnectorResource{client: client}
	var validationDiags diag.Diagnostics
	connectorID := "buffer_validation"
	resource.validateConnector(context.Background(), connector, &connectorID, &validationDiags)
	if validationDiags.HasError() {
		t.Fatalf("validate connector: %v", validationDiags)
	}
	if got := validationDiags.WarningsCount(); got != 1 {
		t.Fatalf("warning count = %d, want 1: %v", got, validationDiags)
	}
	warning := validationDiags[0]
	if warning.Summary() != "Source validation warning: Table Replica Identity" {
		t.Errorf("warning summary = %q", warning.Summary())
	}
	for _, expected := range []string{
		"Some tables require REPLICA IDENTITY FULL.",
		"Suggested action:\nALTER TABLE public.needs_full REPLICA IDENTITY FULL;",
		"Validation test ID: source-config",
	} {
		if !strings.Contains(warning.Detail(), expected) {
			t.Errorf("warning detail does not contain %q: %s", expected, warning.Detail())
		}
	}

	for _, path := range []string{
		"/api/v1/validate/source",
		"/api/v1/validate/buffer",
		"/api/v1/validate/sink",
	} {
		if calls[path] != 1 {
			t.Errorf("%s called %d times; want 1", path, calls[path])
		}
	}
}

func TestConnectorResourceModifyPlanReportsValidationWarnings(t *testing.T) {
	calls := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/validate/source":
			_, _ = w.Write([]byte(`[
  {"test":{"id":"source-config","status":{"Warn":{}},"test_type":{"Config":{"name":"Table Replica Identity","description":"Some tables require REPLICA IDENTITY FULL.","suggestions":["ALTER TABLE public.needs_full REPLICA IDENTITY FULL;"]}},"timestamp":"now"}},
  {"test":{"id":"source-permissions","status":{"Failed":{"reason":"access denied"}},"test_type":{"Permission":{"name":"Replication permissions","description":"Replication permission is missing.","suggestions":["Grant the REPLICATION role."]}},"timestamp":"now"}},
  {"test":{"id":"source-server-info","message":"PostgreSQL version detected.","status":{"Warn":{}},"test_type":{"ServerInfo":{}},"timestamp":"now"}}
]`))
		case "/api/v1/validate/buffer":
			_, _ = w.Write([]byte(`[{"failed":{"reason":"bucket is unavailable"}}]`))
		case "/api/v1/validate/sink":
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatalf("create API client: %v", err)
	}
	model := buildFullConnectorEnvelopeModel()
	model.Disabled = types.BoolValue(false)
	model.Source = &SourceModel{Postgres: buildFullPostgresSourceModel_WithSnapshot()}
	model.Sink = &SinkModel{Duckdb: buildFullDuckDBSinkModel_WithQuack()}

	connectorResource := &ConnectorResource{client: client}
	var schemaResp resource.SchemaResponse
	connectorResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	state := tfsdk.State{
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil),
		Schema: schemaResp.Schema,
	}

	var resp resource.ModifyPlanResponse
	connectorResource.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("plan validation returned errors: %v", resp.Diagnostics)
	}
	if got, want := resp.Diagnostics.WarningsCount(), 5; got != want {
		t.Fatalf("warning count = %d, want %d: %v", got, want, resp.Diagnostics)
	}

	warnings := make([]string, 0, resp.Diagnostics.WarningsCount())
	for _, diagnostic := range resp.Diagnostics {
		warnings = append(warnings, diagnostic.Summary()+"\n"+diagnostic.Detail())
	}
	joined := strings.Join(warnings, "\n")
	for _, expected := range []string{
		"Source validation warning: Table Replica Identity",
		"Some tables require REPLICA IDENTITY FULL.",
		"Suggested action:\nALTER TABLE public.needs_full REPLICA IDENTITY FULL;",
		"Validation test ID: source-config",
		"Source validation failed: Replication permissions",
		"Replication permission is missing.",
		"Failure reason: access denied",
		"Suggested action:\nGrant the REPLICATION role.",
		"Validation test ID: source-permissions",
		"Source validation warning: Server Info",
		"PostgreSQL version detected.",
		"Validation test ID: source-server-info",
		"Buffer validation failed",
		"bucket is unavailable",
		"Sink validation could not run",
		"API returned status 503",
	} {
		if !strings.Contains(joined, expected) {
			t.Errorf("warnings do not contain %q:\n%s", expected, joined)
		}
	}

	for _, path := range []string{
		"/api/v1/validate/source",
		"/api/v1/validate/buffer",
		"/api/v1/validate/sink",
	} {
		if calls[path] != 1 {
			t.Errorf("%s called %d times; want 1", path, calls[path])
		}
	}

	callCount := func() int {
		return calls["/api/v1/validate/source"] + calls["/api/v1/validate/buffer"] + calls["/api/v1/validate/sink"]
	}
	validationPaths := []string{
		"/api/v1/validate/source",
		"/api/v1/validate/buffer",
		"/api/v1/validate/sink",
	}
	assertSkipped := func(t *testing.T, plan tfsdk.Plan, resourceUnderTest *ConnectorResource) {
		t.Helper()
		before := callCount()
		var resp resource.ModifyPlanResponse
		resourceUnderTest.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan, State: state}, &resp)
		if len(resp.Diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		if after := callCount(); after != before {
			t.Fatalf("validation call count changed from %d to %d", before, after)
		}
	}
	assertCalls := func(t *testing.T, plan tfsdk.Plan, expected map[string]int) diag.Diagnostics {
		t.Helper()
		before := make(map[string]int, len(validationPaths))
		for _, validationPath := range validationPaths {
			before[validationPath] = calls[validationPath]
		}
		var resp resource.ModifyPlanResponse
		connectorResource.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan, State: state}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("plan validation returned errors: %v", resp.Diagnostics)
		}
		for _, validationPath := range validationPaths {
			if got := calls[validationPath] - before[validationPath]; got != expected[validationPath] {
				t.Errorf("%s called %d times; want %d", validationPath, got, expected[validationPath])
			}
		}
		return resp.Diagnostics
	}

	t.Run("disabled", func(t *testing.T) {
		disabledModel := *model
		disabledModel.Disabled = types.BoolValue(true)
		disabledPlan := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := disabledPlan.Set(context.Background(), &disabledModel); diags.HasError() {
			t.Fatalf("set disabled plan: %v", diags)
		}
		assertSkipped(t, disabledPlan, connectorResource)
	})

	t.Run("unknown unrelated attribute", func(t *testing.T) {
		unknownPlan := plan
		if diags := unknownPlan.SetAttribute(context.Background(), path.Root("name"), types.StringUnknown()); diags.HasError() {
			t.Fatalf("set unknown plan attribute: %v", diags)
		}
		assertCalls(t, unknownPlan, map[string]int{
			"/api/v1/validate/source": 1,
			"/api/v1/validate/buffer": 1,
			"/api/v1/validate/sink":   1,
		})
	})

	t.Run("unknown sink", func(t *testing.T) {
		unknownPlan := plan
		if diags := unknownPlan.SetAttribute(
			context.Background(),
			path.Root("sink").AtName("duckdb").AtName("connection").AtName("quack").AtName("token"),
			types.StringUnknown(),
		); diags.HasError() {
			t.Fatalf("set unknown sink attribute: %v", diags)
		}
		diagnostics := assertCalls(t, unknownPlan, map[string]int{
			"/api/v1/validate/source": 1,
			"/api/v1/validate/buffer": 1,
		})
		foundReplicaIdentityWarning := false
		for _, diagnostic := range diagnostics {
			if diagnostic.Summary() == "Source validation warning: Table Replica Identity" {
				foundReplicaIdentityWarning = true
				break
			}
		}
		if !foundReplicaIdentityWarning {
			t.Fatalf("missing replica identity warning: %v", diagnostics)
		}
	})

	t.Run("unknown source", func(t *testing.T) {
		unknownPlan := plan
		if diags := unknownPlan.SetAttribute(
			context.Background(),
			path.Root("source").AtName("postgres").AtName("password"),
			types.StringUnknown(),
		); diags.HasError() {
			t.Fatalf("set unknown source attribute: %v", diags)
		}
		assertCalls(t, unknownPlan, map[string]int{
			"/api/v1/validate/buffer": 1,
			"/api/v1/validate/sink":   1,
		})
	})

	t.Run("unknown buffer", func(t *testing.T) {
		unknownPlan := plan
		if diags := unknownPlan.SetAttribute(
			context.Background(),
			path.Root("buffer").AtName("object_store").AtName("url"),
			types.StringUnknown(),
		); diags.HasError() {
			t.Fatalf("set unknown buffer attribute: %v", diags)
		}
		assertCalls(t, unknownPlan, map[string]int{
			"/api/v1/validate/source": 1,
			"/api/v1/validate/sink":   1,
		})
	})

	t.Run("absent buffer", func(t *testing.T) {
		withoutBuffer := *model
		withoutBuffer.Buffer = nil
		withoutBufferPlan := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := withoutBufferPlan.Set(context.Background(), &withoutBuffer); diags.HasError() {
			t.Fatalf("set plan without buffer: %v", diags)
		}
		assertCalls(t, withoutBufferPlan, map[string]int{
			"/api/v1/validate/source": 1,
			"/api/v1/validate/sink":   1,
		})
	})

	t.Run("skip_validation", func(t *testing.T) {
		assertSkipped(t, plan, &ConnectorResource{client: client, skipValidation: true})
	})
}
