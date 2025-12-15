package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	frameworkvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*APIKeyResource)(nil)
var _ resource.ResourceWithConfigure = (*APIKeyResource)(nil)
var _ resource.ResourceWithImportState = (*APIKeyResource)(nil)

// APIKeyResource manages Seq API keys via /api/apikeys.
//
// Ref: https://datalust.co/docs/server-http-api#api-apikeys
type APIKeyResource struct {
	client *Client
}

// APIKeyModel is the Terraform state model for an API key.
type APIKeyModel struct {
	ID                types.String `tfsdk:"id"`
	Title             types.String `tfsdk:"title"`
	Token             types.String `tfsdk:"token"`
	OwnerID           types.String `tfsdk:"owner_id"`
	Permissions       types.Set    `tfsdk:"permissions"`
	MinimumLevel      types.String `tfsdk:"minimum_level"`
	Filter            types.String `tfsdk:"filter"`
	AppliedProperties types.Map    `tfsdk:"applied_properties"`
}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Seq API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Seq API key id.",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "Human-friendly title for the API key.",
				Required:    true,
				Validators: []frameworkvalidator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"token": schema.StringAttribute{
				Description: "The API key token/secret. Seq may only return this on create; it is stored in state as sensitive.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					// If Seq does not return the token on reads, keep the existing value.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.StringAttribute{
				Description: "Owner principal id. Depending on permissions, you may only be able to set this to yourself.",
				Optional:    true,
				Computed:    true,
			},
			"permissions": schema.SetAttribute{
				Description: "Permissions delegated to the API key (e.g. Read, Write, Ingest, Project, System).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"minimum_level": schema.StringAttribute{
				Description: "Minimum log level for events ingested via this API key (e.g. Verbose, Debug, Information, Warning, Error, Fatal). Events below this level will be discarded.",
				Optional:    true,
				Validators: []frameworkvalidator.String{
					stringvalidator.OneOf("Verbose", "Debug", "Information", "Warning", "Error", "Fatal"),
				},
			},
			"filter": schema.StringAttribute{
				Description: "A filter expression to apply to incoming events. Only events matching the filter will be ingested.",
				Optional:    true,
			},
			"applied_properties": schema.MapAttribute{
				Description: "Properties to attach to all events ingested via this API key. These will override any existing properties with the same names.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *provider.Client, got a different type.",
		)
		return
	}
	r.client = client
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var plan APIKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := apiKeyRequestBody(ctx, plan, "AssignedPermissions", "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created apiKeyResponse
	if err := r.client.doJSON(ctx, http.MethodPost, "/api/apikeys", body, &created); err != nil {
		// Back-compat: some Seq versions use "Permissions" instead of "AssignedPermissions".
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest && strings.Contains(httpErr.Message, "AssignedPermissions") {
			legacyBody, legacyDiags := apiKeyRequestBody(ctx, plan, "Permissions", "")
			resp.Diagnostics.Append(legacyDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if err2 := r.client.doJSON(ctx, http.MethodPost, "/api/apikeys", legacyBody, &created); err2 != nil {
				resp.Diagnostics.AddError("Failed to create Seq API key", err2.Error())
				return
			}
		} else {
			resp.Diagnostics.AddError("Failed to create Seq API key", err.Error())
			return
		}
	}

	state := plan
	applyAPIKeyResponse(&state, created)

	// Terraform requires that all values are known (or null) after apply.
	// For Optional+Computed fields, the plan may contain unknown values; if Seq
	// omits a field in the create response, ensure we don't persist unknown.
	if state.OwnerID.IsUnknown() {
		state.OwnerID = types.StringNull()
	}
	if state.Token.IsUnknown() {
		state.Token = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var state APIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.State.RemoveResource(ctx)
		return
	}

	var got apiKeyResponse
	path := "/api/apikeys/" + state.ID.ValueString()
	if err := r.client.doJSON(ctx, http.MethodGet, path, nil, &got); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Seq API key", err.Error())
		return
	}

	newState := state
	applyAPIKeyResponse(&newState, got)

	// Seq may omit token on read; keep previous.
	if got.Token == "" {
		newState.Token = state.Token
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var plan APIKeyModel
	var state APIKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError("Missing id", "Cannot update API key without an id in state")
		return
	}

	apiKeyID := state.ID.ValueString()
	body, diags := apiKeyRequestBody(ctx, plan, "AssignedPermissions", apiKeyID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updated apiKeyResponse
	path := "/api/apikeys/" + apiKeyID
	if err := r.client.doJSON(ctx, http.MethodPut, path, body, &updated); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest && strings.Contains(httpErr.Message, "AssignedPermissions") {
			legacyBody, legacyDiags := apiKeyRequestBody(ctx, plan, "Permissions", apiKeyID)
			resp.Diagnostics.Append(legacyDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if err2 := r.client.doJSON(ctx, http.MethodPut, path, legacyBody, &updated); err2 != nil {
				resp.Diagnostics.AddError("Failed to update Seq API key", err2.Error())
				return
			}
		} else {
			resp.Diagnostics.AddError("Failed to update Seq API key", err.Error())
			return
		}
	}

	newState := plan
	newState.ID = state.ID
	applyAPIKeyResponse(&newState, updated)

	// Token may not be returned on update; keep previous.
	if updated.Token == "" {
		newState.Token = state.Token
	}

	// Ensure Optional+Computed values are not left as unknown after apply.
	if newState.OwnerID.IsUnknown() {
		newState.OwnerID = types.StringNull()
	}
	if newState.Token.IsUnknown() {
		newState.Token = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.checkConfigured(&resp.Diagnostics) {
		return
	}

	var state APIKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		return
	}

	path := "/api/apikeys/" + state.ID.ValueString()
	if err := r.client.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Seq API key", err.Error())
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type apiKeyResponse struct {
	ID      string `json:"Id"`
	Title   string `json:"Title"`
	Token   string `json:"Token"`
	OwnerID string `json:"OwnerId"`
	// Newer Seq versions use AssignedPermissions.
	AssignedPermissions []string `json:"AssignedPermissions"`
	// Older Seq versions use Permissions.
	Permissions   []string           `json:"Permissions"`
	InputSettings *inputSettingsPart `json:"InputSettings"`
}

type inputSettingsPart struct {
	AppliedProperties []eventPropertyPart    `json:"AppliedProperties"`
	Filter            *descriptiveFilterPart `json:"Filter"`
	MinimumLevel      *string                `json:"MinimumLevel"`
}

type eventPropertyPart struct {
	Name  string `json:"Name"`
	Value any    `json:"Value"`
}

type descriptiveFilterPart struct {
	Filter          string `json:"Filter"`
	FilterNonStrict string `json:"FilterNonStrict"`
}

func apiKeyRequestBody(ctx context.Context, plan APIKeyModel, permissionsField string, id string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := map[string]any{
		"Title": plan.Title.ValueString(),
	}

	// Include Id in the body for update operations (Seq requires it to match the URL)
	if id != "" {
		body["Id"] = id
	}

	if !plan.OwnerID.IsNull() && !plan.OwnerID.IsUnknown() && plan.OwnerID.ValueString() != "" {
		body["OwnerId"] = plan.OwnerID.ValueString()
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		diags.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body[permissionsField] = perms
	}

	// Build InputSettings if any of the input settings fields are set
	inputSettings := map[string]any{}

	if !plan.MinimumLevel.IsNull() && !plan.MinimumLevel.IsUnknown() {
		inputSettings["MinimumLevel"] = plan.MinimumLevel.ValueString()
	}

	if !plan.Filter.IsNull() && !plan.Filter.IsUnknown() {
		inputSettings["Filter"] = map[string]any{
			"Filter":          plan.Filter.ValueString(),
			"FilterNonStrict": plan.Filter.ValueString(),
		}
	}

	if !plan.AppliedProperties.IsNull() && !plan.AppliedProperties.IsUnknown() {
		var propsMap map[string]string
		diags.Append(plan.AppliedProperties.ElementsAs(ctx, &propsMap, false)...)
		if diags.HasError() {
			return nil, diags
		}
		var props []map[string]any
		for name, value := range propsMap {
			props = append(props, map[string]any{
				"Name":  name,
				"Value": value,
			})
		}
		inputSettings["AppliedProperties"] = props
	}

	if len(inputSettings) > 0 {
		body["InputSettings"] = inputSettings
	}

	return body, diags
}

func applyAPIKeyResponse(state *APIKeyModel, resp apiKeyResponse) {
	if resp.ID != "" {
		state.ID = types.StringValue(resp.ID)
	}
	if resp.Title != "" {
		state.Title = types.StringValue(resp.Title)
	}
	if resp.Token != "" {
		state.Token = types.StringValue(resp.Token)
	}
	if resp.OwnerID != "" {
		state.OwnerID = types.StringValue(resp.OwnerID)
	}
	perms := resp.AssignedPermissions
	if perms == nil {
		perms = resp.Permissions
	}
	if perms != nil {
		state.Permissions = types.SetValueMust(types.StringType, stringSliceToAttrValues(perms))
	}

	// Apply InputSettings fields
	if resp.InputSettings != nil {
		if resp.InputSettings.MinimumLevel != nil && *resp.InputSettings.MinimumLevel != "" {
			state.MinimumLevel = types.StringValue(*resp.InputSettings.MinimumLevel)
		} else {
			state.MinimumLevel = types.StringNull()
		}

		if resp.InputSettings.Filter != nil && resp.InputSettings.Filter.Filter != "" {
			// Prefer FilterNonStrict if available, otherwise use Filter
			filterValue := resp.InputSettings.Filter.FilterNonStrict
			if filterValue == "" {
				filterValue = resp.InputSettings.Filter.Filter
			}
			state.Filter = types.StringValue(filterValue)
		} else {
			state.Filter = types.StringNull()
		}

		if len(resp.InputSettings.AppliedProperties) > 0 {
			propsMap := make(map[string]attr.Value)
			for _, prop := range resp.InputSettings.AppliedProperties {
				// Convert value to string
				var strValue string
				switch v := prop.Value.(type) {
				case string:
					strValue = v
				default:
					// For non-string values, use JSON representation
					strValue = anyToString(v)
				}
				propsMap[prop.Name] = types.StringValue(strValue)
			}
			state.AppliedProperties = types.MapValueMust(types.StringType, propsMap)
		} else {
			state.AppliedProperties = types.MapNull(types.StringType)
		}
	} else {
		state.MinimumLevel = types.StringNull()
		state.Filter = types.StringNull()
		state.AppliedProperties = types.MapNull(types.StringType)
	}
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON numbers are float64, format without trailing zeros
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func stringSliceToAttrValues(vs []string) []attr.Value {
	out := make([]attr.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, types.StringValue(v))
	}
	return out
}

var errNotConfigured = errors.New("provider not configured")

func (r *APIKeyResource) checkConfigured(respDiags *diag.Diagnostics) bool {
	if r.client == nil {
		respDiags.AddError("Provider not configured", errNotConfigured.Error())
		return false
	}
	return true
}
