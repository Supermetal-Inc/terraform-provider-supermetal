package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

var (
	_ resource.Resource                   = &ConnectorResource{}
	_ resource.ResourceWithImportState    = &ConnectorResource{}
	_ resource.ResourceWithValidateConfig = &ConnectorResource{}

	connectorIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

type ConnectorResource struct {
	client         *api.ClientWithResponses
	skipValidation bool
}

func NewConnectorResource() resource.Resource {
	return &ConnectorResource{}
}

func (r *ConnectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (r *ConnectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Manages a Supermetal CDC connector.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for this connector. Used in the API path and for import. " +
					"Must contain only letters, numbers, hyphens, and underscores (max 30 characters).",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(30),
					stringvalidator.RegexMatches(
						connectorIDPattern,
						"must contain only letters, numbers, hyphens, and underscores",
					),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the connector.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(100),
				},
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether this connector is disabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "Source configuration. Exactly one source type must be specified.",
				Required:            true,
				Attributes:          sourceAttributes(),
			},
			"sink": schema.SingleNestedAttribute{
				MarkdownDescription: "Sink configuration. Exactly one sink type must be specified.",
				Required:            true,
				Attributes:          sinkAttributes(),
			},
		},
	}
}

func (r *ConnectorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	r.client = providerData.Client
	r.skipValidation = providerData.SkipValidation
}

func (r *ConnectorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ConnectorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceCount := countSourceVariants(config.Source)
	if sourceCount == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("source"),
			"Missing source configuration",
			"Exactly one source type must be specified.",
		)
	} else if sourceCount > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("source"),
			"Multiple source types specified",
			"Exactly one source type must be specified.",
		)
	}

	sinkCount := countSinkVariants(config.Sink)
	if sinkCount == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("sink"),
			"Missing sink configuration",
			"Exactly one sink type must be specified.",
		)
	} else if sinkCount > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("sink"),
			"Multiple sink types specified",
			"Exactly one sink type must be specified.",
		)
	}

	if config.Sink != nil && config.Sink.Snowflake != nil && config.Sink.Snowflake.Auth != nil {
		auth := config.Sink.Snowflake.Auth
		if auth.Password != nil && auth.KeyPair != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("sink").AtName("snowflake").AtName("auth"),
				"Multiple Snowflake auth types specified",
				"Exactly one of password or key_pair must be specified.",
			)
		}
		if auth.Password == nil && auth.KeyPair == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("sink").AtName("snowflake").AtName("auth"),
				"Missing Snowflake auth configuration",
				"Exactly one of password or key_pair must be specified.",
			)
		}
	}

	if config.Source != nil {
		if pg := config.Source.Postgres; pg != nil {
			validateTunnel(resp, pg.Tunnel, path.Root("source").AtName("postgres").AtName("tunnel"))
		}
		if mysql := config.Source.Mysql; mysql != nil {
			validateTunnel(resp, mysql.Tunnel, path.Root("source").AtName("mysql").AtName("tunnel"))
		}
	}

	if config.Sink != nil {
		if pg := config.Sink.Postgres; pg != nil {
			validateTunnel(resp, pg.Tunnel, path.Root("sink").AtName("postgres").AtName("tunnel"))
		}
		if duckdb := config.Sink.Duckdb; duckdb != nil && duckdb.Connection != nil && duckdb.Connection.Pg != nil {
			validateTunnel(resp, duckdb.Connection.Pg.Tunnel,
				path.Root("sink").AtName("duckdb").AtName("connection").AtName("pg").AtName("tunnel"))
		}
	}
}

func validateTunnel(resp *resource.ValidateConfigResponse, tunnel *SshTunnelTypeModel, attrPath path.Path) {
	if tunnel != nil && tunnel.Ssh != nil && tunnel.Ssh.Auth != nil {
		validateSSHAuth(resp, tunnel.Ssh.Auth, attrPath.AtName("ssh").AtName("auth"))
	}
}

func validateSSHAuth(resp *resource.ValidateConfigResponse, auth *SshAuthModel, attrPath path.Path) {
	if auth.GeneratedKey != nil && auth.BringYourOwnKey != nil {
		resp.Diagnostics.AddAttributeError(
			attrPath,
			"Multiple SSH auth types specified",
			"Exactly one of generated_key or bring_your_own_key must be specified.",
		)
	}
	if auth.GeneratedKey == nil && auth.BringYourOwnKey == nil {
		resp.Diagnostics.AddAttributeError(
			attrPath,
			"Missing SSH auth configuration",
			"Exactly one of generated_key or bring_your_own_key must be specified.",
		)
	}
}

func (r *ConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connector, diags := plan.toAPIConnector()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.skipValidation && !plan.Disabled.ValueBool() {
		r.validateConnector(ctx, connector, nil, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	apiResp, err := r.client.CreateConnectorWithResponse(ctx, plan.ID.ValueString(), nil, connector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create connector", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"Failed to create connector",
			fmt.Sprintf("API returned status %d: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.GetConnectorWithResponse(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read connector", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Failed to read connector",
			fmt.Sprintf("API returned status %d: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	priorState := state
	state.mergeFromAPI(apiResp.JSON200.Connector, &priorState)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ConnectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connector, diags := plan.toAPIConnector()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.skipValidation && !plan.Disabled.ValueBool() {
		connectorID := plan.ID.ValueString()
		r.validateConnector(ctx, connector, &connectorID, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	apiResp, err := r.client.CreateConnectorWithResponse(ctx, plan.ID.ValueString(), nil, connector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update connector", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"Failed to update connector",
			fmt.Sprintf("API returned status %d: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorID := state.ID.ValueString()

	if r.sendDeleteAndPoll(ctx, connectorID, api.Delete, 30*time.Second, resp) {
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	r.emitGracefulDeleteWarning(ctx, connectorID, &state, resp)

	if r.sendDeleteAndPoll(ctx, connectorID, api.ForceDelete, 30*time.Second, resp) {
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError(
		"Connector force deletion timed out",
		fmt.Sprintf("Connector %q did not reach 404 after ForceDelete. Manual cleanup may be required.", connectorID),
	)
}

func (r *ConnectorResource) sendDeleteAndPoll(ctx context.Context, id string, cmd api.ConnectorStatusCommand, timeout time.Duration, resp *resource.DeleteResponse) bool {
	cmdResp, err := r.client.SendConnectorCommandWithResponse(ctx, id, api.CommandWrapper{Command: cmd})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to send %s command", cmd),
			err.Error(),
		)
		return false
	}
	if cmdResp.StatusCode() == http.StatusNotFound {
		return true
	}
	if cmdResp.StatusCode() != http.StatusOK && cmdResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to send %s command", cmd),
			fmt.Sprintf("API returned status %d: %s", cmdResp.StatusCode(), string(cmdResp.Body)),
		)
		return false
	}

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		checkResp, err := r.client.GetConnectorWithResponse(pollCtx, id)
		if err != nil {
			if ctx.Err() != nil {
				resp.Diagnostics.AddError(
					"Connector deletion cancelled",
					"The operation was cancelled before confirming the connector was deleted.",
				)
				return false
			}
			if pollCtx.Err() != nil {
				return false
			}
			resp.Diagnostics.AddError("Failed to poll connector deletion", err.Error())
			return false
		}
		if checkResp.StatusCode() == http.StatusNotFound {
			return true
		}
		select {
		case <-pollCtx.Done():
			if ctx.Err() != nil {
				resp.Diagnostics.AddError(
					"Connector deletion cancelled",
					"The operation was cancelled before confirming the connector was deleted.",
				)
				return false
			}
			return false
		case <-ticker.C:
		}
	}
}

func (r *ConnectorResource) emitGracefulDeleteWarning(ctx context.Context, id string, state *ConnectorModel, resp *resource.DeleteResponse) {
	var reason string
	getResp, err := r.client.GetConnectorWithResponse(ctx, id)
	if err != nil {
		tflog.Warn(ctx, "failed to fetch connector status during delete", map[string]any{
			"error": err.Error(),
		})
	} else if getResp.JSON200 != nil && getResp.JSON200.Status != nil {
		reason = extractFailureReason(getResp.JSON200.Status)
	}

	var cleanupHint string
	if state.Source != nil && state.Source.Postgres != nil {
		pg := state.Source.Postgres
		host := pg.Host.ValueString()
		database := pg.Database.ValueString()
		var publication string
		if pg.ReplicationType != nil && pg.ReplicationType.LogicalReplication != nil {
			publication = pg.ReplicationType.LogicalReplication.PublicationName.ValueString()
		}
		if publication == "" {
			publication = "supermetal_" + id
		}
		slotName := "supermetal_" + strings.ReplaceAll(id, "-", "_")
		cleanupHint = fmt.Sprintf(
			"A replication slot may remain on the PostgreSQL server. "+
				"If the source is still accessible, run:\n\n"+
				"  SELECT pg_drop_replication_slot('%s');\n  DROP PUBLICATION IF EXISTS \"%s\";\n\n"+
				"on %s/%s. Orphaned slots cause WAL retention to grow unbounded.",
			slotName, publication, host, database,
		)
	}

	detail := "Graceful Delete timed out. Escalating to ForceDelete."
	if reason != "" {
		detail = fmt.Sprintf("Graceful Delete failed: %s. Escalating to ForceDelete.", reason)
	}
	if cleanupHint != "" {
		detail = detail + "\n\n" + cleanupHint
	}

	resp.Diagnostics.AddWarning(
		fmt.Sprintf("Connector %q required force deletion", id),
		detail,
	)
}

func extractFailureReason(status *api.ConnectorStatusConnectorStatus) string {
	if status == nil {
		return ""
	}
	failed, err := status.Status.AsConnectorStatusStatus2()
	if err == nil && failed.Failed.Reason != nil {
		return *failed.Failed.Reason
	}
	return ""
}

func (r *ConnectorResource) validateConnector(ctx context.Context, connector api.ConnectorConnector, connectorID *string, diags *diag.Diagnostics) {
	sourceParams := &api.ValidateSourceParams{ConnectorId: connectorID}
	sourceBody := api.ValidateSourceJSONRequestBody{Source: connector.Source}
	sourceResp, err := r.client.ValidateSourceWithResponse(ctx, sourceParams, sourceBody)
	if err != nil {
		diags.AddError("Source validation failed", err.Error())
		return
	}
	if sourceResp.StatusCode() != http.StatusOK {
		diags.AddError(
			"Source validation failed",
			fmt.Sprintf("API returned status %d: %s", sourceResp.StatusCode(), string(sourceResp.Body)),
		)
		return
	}
	if reason := extractValidationFailure(sourceResp.JSON200); reason != "" {
		diags.AddError("Source validation failed", reason)
		return
	}

	sinkParams := &api.ValidateSinkParams{ConnectorId: connectorID}
	sinkBody := api.ValidateSinkJSONRequestBody{Sink: connector.Sink}
	sinkResp, err := r.client.ValidateSinkWithResponse(ctx, sinkParams, sinkBody)
	if err != nil {
		diags.AddError("Sink validation failed", err.Error())
		return
	}
	if sinkResp.StatusCode() != http.StatusOK {
		diags.AddError(
			"Sink validation failed",
			fmt.Sprintf("API returned status %d: %s", sinkResp.StatusCode(), string(sinkResp.Body)),
		)
		return
	}
	if reason := extractValidationFailure(sinkResp.JSON200); reason != "" {
		diags.AddError("Sink validation failed", reason)
		return
	}
}

func extractValidationFailure(events *[]api.ConnectorValidateEvent) string {
	if events == nil {
		return ""
	}
	reasons := make([]string, 0, len(*events))
	for _, event := range *events {
		failed, err := event.AsConnectorValidateEvent2()
		if err == nil && failed.Failed.Reason != nil {
			reasons = append(reasons, *failed.Failed.Reason)
			continue
		}
		test, err := event.AsConnectorValidateEvent1()
		if err == nil {
			failedStatus, err := test.Test.Status.AsConnectorValidateStatus2()
			if err == nil && failedStatus.Failed.Reason != nil {
				reasons = append(reasons, *failedStatus.Failed.Reason)
			}
		}
	}
	return strings.Join(reasons, "; ")
}

func (r *ConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
