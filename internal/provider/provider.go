package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

var _ provider.Provider = &SupermetalProvider{}

type SupermetalProvider struct {
	version string
}

type SupermetalProviderModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	CACert         types.String `tfsdk:"ca_cert"`
	Insecure       types.Bool   `tfsdk:"insecure"`
	SkipValidation types.Bool   `tfsdk:"skip_validation"`
}

type ProviderData struct {
	Client         *api.ClientWithResponses
	SkipValidation bool
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SupermetalProvider{version: version}
	}
}

func (p *SupermetalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "supermetal"
	resp.Version = p.version
}

func (p *SupermetalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Supermetal provider manages CDC connectors on Supermetal agents.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Supermetal agent API endpoint (e.g. `https://sm.internal:3000`). " +
					"Can also be set via `SUPERMETAL_ENDPOINT` environment variable.",
				Optional: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for basic authentication. " +
					"Can also be set via `SUPERMETAL_USERNAME` environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for basic authentication. " +
					"Can also be set via `SUPERMETAL_PASSWORD` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"ca_cert": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded CA certificate for TLS verification. " +
					"Can also be set via `SUPERMETAL_CA_CERT` environment variable.",
				Optional: true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification. Not recommended for production.",
				Optional:            true,
			},
			"skip_validation": schema.BoolAttribute{
				MarkdownDescription: "Skip server-side validation of source and sink configurations before create/update. " +
					"Can also be set via `SUPERMETAL_SKIP_VALIDATION` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *SupermetalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config SupermetalProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown endpoint",
			"The endpoint cannot be determined until apply. Set it explicitly or via SUPERMETAL_ENDPOINT.",
		)
		return
	}

	endpoint := getConfigValue(config.Endpoint, "SUPERMETAL_ENDPOINT")
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing endpoint",
			"The Supermetal agent endpoint must be set via the endpoint attribute or SUPERMETAL_ENDPOINT environment variable.",
		)
		return
	}

	username := getConfigValue(config.Username, "SUPERMETAL_USERNAME")
	password := getConfigValue(config.Password, "SUPERMETAL_PASSWORD")
	caCert := getConfigValue(config.CACert, "SUPERMETAL_CA_CERT")
	insecure := config.Insecure.ValueBool()
	skipValidation := getBoolConfigValue(config.SkipValidation, "SUPERMETAL_SKIP_VALIDATION")

	if insecure {
		tflog.Warn(ctx, "TLS certificate verification disabled")
	}

	if (username == "") != (password == "") {
		tflog.Warn(ctx, "only one of username/password is set, basic auth will not be configured")
	}

	httpClient, err := buildHTTPClient(caCert, insecure)
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure HTTP client", err.Error())
		return
	}

	var opts []api.ClientOption
	opts = append(opts, api.WithHTTPClient(httpClient))
	if username != "" && password != "" {
		opts = append(opts, api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.SetBasicAuth(username, password)
			return nil
		}))
	}
	opts = append(opts, api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", "terraform-provider-supermetal/"+p.version)
		return nil
	}))

	client, err := api.NewClientWithResponses(endpoint, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Supermetal client", err.Error())
		return
	}

	providerData := &ProviderData{
		Client:         client,
		SkipValidation: skipValidation,
	}
	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *SupermetalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewConnectorResource,
	}
}

func (p *SupermetalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func getConfigValue(tfValue types.String, envVar string) string {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueString()
	}
	return os.Getenv(envVar)
}

func getBoolConfigValue(tfValue types.Bool, envVar string) bool {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueBool()
	}
	v, _ := strconv.ParseBool(os.Getenv(envVar))
	return v
}

func buildHTTPClient(caCert string, insecure bool) (*http.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}

	if caCert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caCert)) {
			return nil, errors.New("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = pool
	}

	baseTransport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 10,
	}

	return &http.Client{
		Transport: &retryTransport{
			base:        baseTransport,
			maxAttempts: 3,
			baseDelay:   100 * time.Millisecond,
		},
		Timeout: 30 * time.Second,
	}, nil
}

type retryTransport struct {
	base        http.RoundTripper
	maxAttempts int
	baseDelay   time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= t.maxAttempts; attempt++ {
		if attempt > 1 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}
		resp, err := t.base.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		if !isConnectionError(err) {
			return nil, err
		}
		lastErr = err
		if attempt < t.maxAttempts {
			delay := t.baseDelay * time.Duration(1<<(attempt-1))
			tflog.Warn(req.Context(), "connection error, retrying",
				map[string]any{
					"attempt": attempt,
					"max":     t.maxAttempts,
					"delay":   delay.String(),
					"error":   err.Error(),
				})
			timer := time.NewTimer(delay)
			select {
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			case <-timer.C:
			}
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", t.maxAttempts, lastErr)
}

func isConnectionError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
