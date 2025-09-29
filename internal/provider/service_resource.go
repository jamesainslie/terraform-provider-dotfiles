// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/platform"
	"github.com/jamesainslie/terraform-provider-dotfiles/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *DotfilesClient
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	DesiredState  types.String `tfsdk:"desired_state"`
	Scope         types.String `tfsdk:"scope"`
	RestartMethod types.String `tfsdk:"restart_method"`
	Timeout       types.String `tfsdk:"timeout"`
	OnlyIf        *OnlyIfModel `tfsdk:"only_if"`

	// Computed attributes for state tracking
	ActualState   types.String `tfsdk:"actual_state"`
	LastOperation types.String `tfsdk:"last_operation"`
	LastApplied   types.String `tfsdk:"last_applied"`
}

// OnlyIfModel defines conditional execution rules
type OnlyIfModel struct {
	ServiceExists types.Bool   `tfsdk:"service_exists"`
	FileChanged   types.String `tfsdk:"file_changed"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages system service state declaratively. Replaces shell commands like 'systemctl restart nginx' with native Go service management.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service resource identifier",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Service name to manage",
				Validators: []validator.String{
					validators.NotEmpty(),
				},
			},
			"desired_state": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Desired service state: running, stopped, restarted, reloaded",
				Validators: []validator.String{
					validators.OneOf("running", "stopped", "restarted", "reloaded"),
				},
			},
			"scope": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("user"),
				MarkdownDescription: "Service scope: user (default) or system",
				Validators: []validator.String{
					validators.OneOf("user", "system"),
				},
			},
			"restart_method": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("graceful"),
				MarkdownDescription: "How to restart the service: graceful (default), force, signal",
				Validators: []validator.String{
					validators.OneOf("graceful", "force", "signal"),
				},
			},
			"timeout": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("30s"),
				MarkdownDescription: "Timeout for service operations (default: 30s)",
			},
			"actual_state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current actual state of the service",
			},
			"last_operation": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last operation performed on the service",
			},
			"last_applied": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of last successful operation",
			},
		},

		Blocks: map[string]schema.Block{
			"only_if": schema.SingleNestedBlock{
				MarkdownDescription: "Conditions for when to manage the service",
				Attributes: map[string]schema.Attribute{
					"service_exists": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Only manage if service exists (default: true)",
					},
					"file_changed": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Only manage if file hash changed (content hash to monitor)",
					},
				},
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *DotfilesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating service resource", map[string]interface{}{
		"name":          data.Name.ValueString(),
		"desired_state": data.DesiredState.ValueString(),
		"scope":         data.Scope.ValueString(),
	})

	// Apply the service state
	if err := r.applyServiceState(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Service Operation Failed",
			fmt.Sprintf("Unable to apply service state: %s", err.Error()),
		)
		return
	}

	// Generate ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Name.ValueString(), data.Scope.ValueString()))
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading service resource", map[string]interface{}{
		"name": data.Name.ValueString(),
		"id":   data.ID.ValueString(),
	})

	// Get current service state
	actualState, err := r.getActualServiceState(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Service State Read Failed",
			fmt.Sprintf("Unable to read service state: %s", err.Error()),
		)
		return
	}

	data.ActualState = types.StringValue(actualState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating service resource", map[string]interface{}{
		"name":          data.Name.ValueString(),
		"desired_state": data.DesiredState.ValueString(),
	})

	// Apply the service state changes
	if err := r.applyServiceState(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Service Operation Failed",
			fmt.Sprintf("Unable to update service state: %s", err.Error()),
		)
		return
	}

	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting service resource", map[string]interface{}{
		"name": data.Name.ValueString(),
		"id":   data.ID.ValueString(),
	})

	// For service resources, deletion typically means leaving the service in its current state
	// We don't actively stop or start services on resource deletion unless explicitly configured
	tflog.Info(ctx, "Service resource deleted - service state unchanged", map[string]interface{}{
		"service": data.Name.ValueString(),
	})
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID (format: "service_name:scope")
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID in format 'service_name:scope' (e.g., 'nginx:system')",
		)
		return
	}

	serviceName := parts[0]
	scope := parts[1]

	// Validate scope
	if scope != "user" && scope != "system" {
		resp.Diagnostics.AddError(
			"Invalid Scope",
			fmt.Sprintf("Scope must be 'user' or 'system', got: %s", scope),
		)
		return
	}

	// Create a basic service resource model for import
	importData := ServiceResourceModel{
		ID:            types.StringValue(req.ID),
		Name:          types.StringValue(serviceName),
		Scope:         types.StringValue(scope),
		DesiredState:  types.StringValue("running"), // Default state for import
		RestartMethod: types.StringValue("graceful"),
		Timeout:       types.StringValue("30s"),
		ActualState:   types.StringNull(),
		LastOperation: types.StringNull(),
		LastApplied:   types.StringNull(),
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &importData)...)
}

// applyServiceState applies the desired service state using native platform operations
func (r *ServiceResource) applyServiceState(ctx context.Context, data *ServiceResourceModel) error {
	platformProvider := platform.DetectPlatform()
	extProvider, ok := platformProvider.(platform.ExtendedPlatformProvider)
	if !ok {
		return fmt.Errorf("platform does not support service management")
	}

	serviceManager := extProvider.ServiceManager()
	serviceName := data.Name.ValueString()
	userLevel := data.Scope.ValueString() == "user"
	desiredState := data.DesiredState.ValueString()

	// Check conditions if specified
	if data.OnlyIf != nil {
		if !data.OnlyIf.ServiceExists.IsNull() && data.OnlyIf.ServiceExists.ValueBool() {
			if !serviceManager.ServiceExists(serviceName, userLevel) {
				return fmt.Errorf("service %s does not exist in %s scope", serviceName, data.Scope.ValueString())
			}
		}

		// TODO: Implement file_changed condition when file resource integration is complete
		if !data.OnlyIf.FileChanged.IsNull() {
			tflog.Warn(ctx, "file_changed condition not yet implemented", map[string]interface{}{
				"service": serviceName,
			})
		}
	}

	// Parse timeout
	timeout := 30 * time.Second
	if !data.Timeout.IsNull() {
		if t, err := time.ParseDuration(data.Timeout.ValueString()); err == nil {
			timeout = t
		}
	}

	// Create context with timeout for service operations
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var err error
	var operation string

	// Apply the desired state
	switch desiredState {
	case "running":
		operation = "start"
		err = serviceManager.StartService(serviceName, userLevel)
	case "stopped":
		operation = "stop"
		err = serviceManager.StopService(serviceName, userLevel)
	case "restarted":
		operation = "restart"
		err = serviceManager.RestartService(serviceName, userLevel)
	case "reloaded":
		operation = "reload"
		err = serviceManager.ReloadService(serviceName, userLevel)
	default:
		return fmt.Errorf("unsupported desired state: %s", desiredState)
	}

	if err != nil {
		return fmt.Errorf("failed to %s service %s: %w", operation, serviceName, err)
	}

	// Update computed attributes
	data.LastOperation = types.StringValue(operation)

	// Get current state after operation
	actualState, err := r.getActualServiceState(operationCtx, data)
	if err != nil {
		tflog.Warn(ctx, "Failed to get actual state after operation", map[string]interface{}{
			"error": err.Error(),
		})
		// Don't fail the operation just because we can't read the state
	} else {
		data.ActualState = types.StringValue(actualState)
	}

	tflog.Info(ctx, "Service operation completed successfully", map[string]interface{}{
		"service":   serviceName,
		"operation": operation,
		"scope":     data.Scope.ValueString(),
	})

	return nil
}

// getActualServiceState retrieves the current state of the service
func (r *ServiceResource) getActualServiceState(_ context.Context, data *ServiceResourceModel) (string, error) {
	platformProvider := platform.DetectPlatform()
	extProvider, ok := platformProvider.(platform.ExtendedPlatformProvider)
	if !ok {
		return "", fmt.Errorf("platform does not support service management")
	}

	serviceManager := extProvider.ServiceManager()
	serviceName := data.Name.ValueString()
	userLevel := data.Scope.ValueString() == "user"

	if !serviceManager.ServiceExists(serviceName, userLevel) {
		return "not_found", nil
	}

	status, err := serviceManager.GetServiceStatus(serviceName, userLevel)
	if err != nil {
		return "", fmt.Errorf("failed to get service status: %w", err)
	}

	return status.State, nil
}
