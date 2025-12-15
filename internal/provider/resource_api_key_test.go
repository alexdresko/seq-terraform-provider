package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestClientAddsAPIKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Seq-ApiKey")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := &Client{baseURL: mustParseURL(srv.URL), apiKey: "abc", http: srv.Client()}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if got != "abc" {
		t.Fatalf("expected X-Seq-ApiKey header to be set, got %q", got)
	}
}

func TestAPIKeyRequestBody(t *testing.T) {
	m := APIKeyModel{
		Title:       types.StringValue("x"),
		OwnerID:     types.StringValue("owner"),
		Permissions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Read"), types.StringValue("Write")}),
	}
	body, diags := apiKeyRequestBody(context.Background(), m, "AssignedPermissions", "")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics")
	}
	if body["Title"].(string) != "x" {
		t.Fatalf("title mismatch")
	}
	if _, ok := body["AssignedPermissions"]; !ok {
		t.Fatalf("expected AssignedPermissions in request body")
	}
	// ID should not be present when empty
	if _, ok := body["Id"]; ok {
		t.Fatalf("expected Id to be absent for create operations")
	}
}

func TestAPIKeyRequestBodyWithID(t *testing.T) {
	m := APIKeyModel{
		Title:       types.StringValue("x"),
		OwnerID:     types.StringValue("owner"),
		Permissions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Read")}),
	}
	body, diags := apiKeyRequestBody(context.Background(), m, "AssignedPermissions", "apikey-123")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics")
	}
	// ID should be present for update operations
	if id, ok := body["Id"].(string); !ok || id != "apikey-123" {
		t.Fatalf("expected Id 'apikey-123' in request body for update, got %v", body["Id"])
	}
}

func TestAPIKeyRequestBodyWithInputSettings(t *testing.T) {
	m := APIKeyModel{
		Title:        types.StringValue("test-key"),
		OwnerID:      types.StringNull(),
		Permissions:  types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Ingest")}),
		MinimumLevel: types.StringValue("Warning"),
		Filter:       types.StringValue("@Level = 'Error'"),
		AppliedProperties: types.MapValueMust(types.StringType, map[string]attr.Value{
			"Application": types.StringValue("MyApp"),
			"Environment": types.StringValue("Production"),
		}),
	}
	body, diags := apiKeyRequestBody(context.Background(), m, "AssignedPermissions", "")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	inputSettings, ok := body["InputSettings"].(map[string]any)
	if !ok {
		t.Fatalf("expected InputSettings in request body")
	}

	if inputSettings["MinimumLevel"] != "Warning" {
		t.Fatalf("expected MinimumLevel to be 'Warning', got %v", inputSettings["MinimumLevel"])
	}

	filter, ok := inputSettings["Filter"].(map[string]any)
	if !ok {
		t.Fatalf("expected Filter in InputSettings")
	}
	if filter["Filter"] != "@Level = 'Error'" {
		t.Fatalf("expected Filter to be '@Level = 'Error'', got %v", filter["Filter"])
	}

	props, ok := inputSettings["AppliedProperties"].([]map[string]any)
	if !ok {
		t.Fatalf("expected AppliedProperties in InputSettings")
	}
	if len(props) != 2 {
		t.Fatalf("expected 2 applied properties, got %d", len(props))
	}
}

func TestApplyAPIKeyResponseWithInputSettings(t *testing.T) {
	minLevel := "Error"
	resp := apiKeyResponse{
		ID:                  "apikey-123",
		Title:               "Test Key",
		Token:               "secret-token",
		OwnerID:             "user-1",
		AssignedPermissions: []string{"Ingest", "Read"},
		InputSettings: &inputSettingsPart{
			MinimumLevel: &minLevel,
			Filter: &descriptiveFilterPart{
				Filter:          "@Level = 'Error'",
				FilterNonStrict: "Level = Error",
			},
			AppliedProperties: []eventPropertyPart{
				{Name: "Application", Value: "TestApp"},
				{Name: "Version", Value: "1.0"},
			},
		},
	}

	state := &APIKeyModel{}
	applyAPIKeyResponse(state, resp)

	if state.ID.ValueString() != "apikey-123" {
		t.Fatalf("expected ID 'apikey-123', got %q", state.ID.ValueString())
	}
	if state.MinimumLevel.ValueString() != "Error" {
		t.Fatalf("expected MinimumLevel 'Error', got %q", state.MinimumLevel.ValueString())
	}
	// Should prefer FilterNonStrict
	if state.Filter.ValueString() != "Level = Error" {
		t.Fatalf("expected Filter 'Level = Error', got %q", state.Filter.ValueString())
	}
	if state.AppliedProperties.IsNull() {
		t.Fatalf("expected AppliedProperties to not be null")
	}

	var propsMap map[string]string
	diags := state.AppliedProperties.ElementsAs(context.Background(), &propsMap, false)
	if diags.HasError() {
		t.Fatalf("failed to get AppliedProperties: %v", diags)
	}
	if propsMap["Application"] != "TestApp" {
		t.Fatalf("expected Application 'TestApp', got %q", propsMap["Application"])
	}
}

func TestApplyAPIKeyResponseWithEmptyInputSettings(t *testing.T) {
	resp := apiKeyResponse{
		ID:                  "apikey-456",
		Title:               "Simple Key",
		AssignedPermissions: []string{"Ingest"},
		InputSettings:       nil,
	}

	state := &APIKeyModel{}
	applyAPIKeyResponse(state, resp)

	if !state.MinimumLevel.IsNull() {
		t.Fatalf("expected MinimumLevel to be null, got %q", state.MinimumLevel.ValueString())
	}
	if !state.Filter.IsNull() {
		t.Fatalf("expected Filter to be null, got %q", state.Filter.ValueString())
	}
	if !state.AppliedProperties.IsNull() {
		t.Fatalf("expected AppliedProperties to be null")
	}
}
