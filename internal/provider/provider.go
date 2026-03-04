package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/legioner0/go-icinga2-api/iapi"
)

var (
	_ provider.Provider = &icinga2Provider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &icinga2Provider{
			version: version,
		}
	}
}

type icinga2Provider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type icinga2ProviderModel struct {
	Host                     types.String `tfsdk:"api_url"`
	Username                 types.String `tfsdk:"api_user"`
	Password                 types.String `tfsdk:"api_password"`
	Insecure_skip_tls_verify types.Bool   `tfsdk:"insecure_skip_tls_verify"`
	Ca_cert_file             types.String `tfsdk:"ca_cert_file"`
	Retries                  types.Int32  `tfsdk:"retries"`
	Retry_delay              types.String `tfsdk:"retry_delay"`
}

func (p *icinga2Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "icinga2"
	resp.Version = p.version
}

func (p *icinga2Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The address of the Icinga2 server.",
			},
			"api_user": schema.StringAttribute{
				Optional:    true,
				Description: "The user to authenticate to the Icinga2 Server as.",
			},
			"api_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The password for authenticating to the Icinga2 server.",
			},
			"insecure_skip_tls_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Disable TLS verify when connecting to Icinga2 Server.",
			},
			"ca_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The CA certificate of Icinga2 Server.",
			},
			"retries": schema.Int32Attribute{
				Optional:    true,
				Description: "How many times to retry on low level errors and `503 Icinga is reloading`. Defaults to `0`.",
			},
			"retry_delay": schema.StringAttribute{
				Optional:    true,
				Description: "Delay between retry attempts. Valid values are durations expressed as `500ms`, etc. or a plain number which is treated as whole seconds.",
			},
		},
	}
}

func (p *icinga2Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config icinga2ProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown icinga2 API Host",
			"The provider cannot create the icinga2 API client as there is an unknown configuration value for the icinga2 API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ICINGA2_API_URL environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_user"),
			"Unknown icinga2 API Username",
			"The provider cannot create the icinga2 API client as there is an unknown configuration value for the icinga2 API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ICINGA2_API_USER environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_password"),
			"Unknown icinga2 API Password",
			"The provider cannot create the icinga2 API client as there is an unknown configuration value for the icinga2 API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ICINGA2_API_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	api_url := os.Getenv("ICINGA2_API_URL")
	api_user := os.Getenv("ICINGA2_API_USER")
	api_password := os.Getenv("ICINGA2_API_PASSWORD")
	tlsVerify, _ := strconv.ParseBool(os.Getenv("ICINGA2_INSECURE_SKIP_TLS_VERIFY"))
	ca_cert_file := os.Getenv("ICINGA2_API_CA_CERT_FILE")
	retries64, _ := strconv.ParseInt(os.Getenv("ICINGA2_API_RETRIES"), 10, 32)
	retries := int32(retries64)
	retry_delay := os.Getenv("ICINGA2_API_RETRY_DELAY")

	if !config.Host.IsNull() {
		api_url = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		api_user = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		api_password = config.Password.ValueString()
	}

	if !config.Ca_cert_file.IsNull() {
		ca_cert_file = config.Ca_cert_file.ValueString()
	}

	if !config.Retries.IsNull() {
		retries = config.Retries.ValueInt32()
	}

	if !config.Retry_delay.IsNull() {
		retry_delay = config.Retry_delay.ValueString()
	}

	if api_url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing icinga2 API Host",
			"The provider cannot create the icinga2 API client as there is a missing or empty value for the icinga2 API host. "+
				"Set the host value in the configuration or use the ICINGA2_API_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if api_user == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_user"),
			"Missing icinga2 API Username",
			"The provider cannot create the icinga2 API client as there is a missing or empty value for the icinga2 API username. "+
				"Set the username value in the configuration or use the ICINGA2_API_USER environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if api_password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing icinga2 API Password",
			"The provider cannot create the icinga2 API client as there is a missing or empty value for the icinga2 API password. "+
				"Set the password value in the configuration or use the ICINGA2_API_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if ca_cert_file != "" {
		_, err := os.Stat(ca_cert_file)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("ca_cert_file"),
				"Invalid value for CA certificate of Icinga2 Server",
				"The provider cannot create the icinga2 API client as there is a missing CA certificate of Icinga2 Server. "+
					"Set the ca_cert_file value in the configuration or use the ICINGA2_API_CA_CERT_FILE environment variable. "+
					"If either is already set, ensure the value is not empty and file exist",
			)
		}
	}

	var duration time.Duration
	if retry_delay != "" {
		var err error
		// Try parsing as a duration
		duration, err = time.ParseDuration(retry_delay)
		if err != nil {
			// Failing that, convert to an integer and treat as seconds
			seconds, err := strconv.Atoi(retry_delay)
			if err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("retry_delay"),
					"Invalid icinga2 API retry delay",
					"The provider cannot create the icinga2 API client as there is an invalid configuration value for the icinga2 API retry delay. "+
						"Either target apply the source of the value first, set the value statically in the configuration, or use the ICINGA2_API_RETRY_DELAY environment variable.",
				)
			}
			duration = time.Duration(seconds) * time.Second
		}
		if duration < 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("retry_delay"),
				"Invalid icinga2 API retry delay",
				"The provider cannot create the icinga2 API client as there is an invalid configuration value for the icinga2 API retry delay. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the ICINGA2_API_RETRY_DELAY environment variable.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := iapi.New(
		api_user,
		api_password,
		api_url,
		tlsVerify,
		ca_cert_file,
		retries,
		duration,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create icinga2 API Client",
			"An unexpected error occurred when creating the icinga2 API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"icinga2 Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *icinga2Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *icinga2Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		HostGroup,
	}
}
