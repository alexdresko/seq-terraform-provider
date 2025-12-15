package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*HealthDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*HealthDataSource)(nil)

// HealthDataSource reads /health from the Seq server.
//
// Ref: https://datalust.co/docs/server-http-api#health
type HealthDataSource struct {
	client *Client
}

type HealthModel struct {
	Status types.String `tfsdk:"status"`
}

func NewHealthDataSource() datasource.DataSource {
	return &HealthDataSource{}
}

func (d *HealthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *HealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the Seq /health endpoint.",
		Attributes: map[string]schema.Attribute{
			"status": schema.StringAttribute{
				Description: "Health status message returned by Seq.",
				Computed:    true,
			},
		},
	}
}

func (d *HealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *provider.Client, got a different type.",
		)
		return
	}
	d.client = client
}

func (d *HealthDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Missing configured Seq client")
		return
	}

	var body map[string]any
	if err := d.client.doJSON(ctx, http.MethodGet, "/health", nil, &body); err != nil {
		resp.Diagnostics.AddError("Failed to read Seq /health", err.Error())
		return
	}

	status := ""
	if v, ok := body["status"].(string); ok {
		status = v
	}

	state := HealthModel{Status: types.StringValue(status)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
