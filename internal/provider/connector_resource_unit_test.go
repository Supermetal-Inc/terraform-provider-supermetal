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

func TestConnectorResourceValidateConnectorCallsBuffer(t *testing.T) {
	calls := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
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
	connectorID := "buffer-validation"
	resource.validateConnector(context.Background(), connector, &connectorID, &validationDiags)
	if validationDiags.HasError() {
		t.Fatalf("validate connector: %v", validationDiags)
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

	t.Run("disabled", func(t *testing.T) {
		disabledModel := *model
		disabledModel.Disabled = types.BoolValue(true)
		disabledPlan := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := disabledPlan.Set(context.Background(), &disabledModel); diags.HasError() {
			t.Fatalf("set disabled plan: %v", diags)
		}
		assertSkipped(t, disabledPlan, connectorResource)
	})

	t.Run("unknown", func(t *testing.T) {
		unknownPlan := plan
		if diags := unknownPlan.SetAttribute(context.Background(), path.Root("name"), types.StringUnknown()); diags.HasError() {
			t.Fatalf("set unknown plan attribute: %v", diags)
		}
		assertSkipped(t, unknownPlan, connectorResource)
	})

	t.Run("skip_validation", func(t *testing.T) {
		assertSkipped(t, plan, &ConnectorResource{client: client, skipValidation: true})
	})
}
