package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
