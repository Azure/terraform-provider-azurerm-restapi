package services

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Azure/terraform-provider-azapi/internal/clients"
	"github.com/Azure/terraform-provider-azapi/internal/docstrings"
	"github.com/Azure/terraform-provider-azapi/internal/locks"
	"github.com/Azure/terraform-provider-azapi/internal/retry"
	"github.com/Azure/terraform-provider-azapi/internal/services/defaults"
	"github.com/Azure/terraform-provider-azapi/internal/services/dynamic"
	"github.com/Azure/terraform-provider-azapi/internal/services/migration"
	"github.com/Azure/terraform-provider-azapi/internal/services/myplanmodifier"
	"github.com/Azure/terraform-provider-azapi/internal/services/myvalidator"
	"github.com/Azure/terraform-provider-azapi/internal/services/parse"
	"github.com/Azure/terraform-provider-azapi/internal/skip"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ActionResourceModel struct {
	ID                            types.String     `tfsdk:"id"`
	Type                          types.String     `tfsdk:"type"`
	ResourceId                    types.String     `tfsdk:"resource_id"`
	Action                        types.String     `tfsdk:"action"`
	Method                        types.String     `tfsdk:"method"`
	Body                          types.Dynamic    `tfsdk:"body"`
	When                          types.String     `tfsdk:"when"`
	Locks                         types.List       `tfsdk:"locks"`
	ResponseExportValues          types.Dynamic    `tfsdk:"response_export_values"`
	SensitiveResponseExportValues types.Dynamic    `tfsdk:"sensitive_response_export_values"`
	Output                        types.Dynamic    `tfsdk:"output"`
	SensitiveOutput               types.Dynamic    `tfsdk:"sensitive_output"`
	Timeouts                      timeouts.Value   `tfsdk:"timeouts" skip_on:"update"`
	Retry                         retry.RetryValue `tfsdk:"retry" skip_on:"update"`
	Headers                       types.Map        `tfsdk:"headers"`
	QueryParameters               types.Map        `tfsdk:"query_parameters"`
}

type ActionResource struct {
	ProviderData *clients.Client
}

var _ resource.Resource = &ActionResource{}
var _ resource.ResourceWithConfigure = &ActionResource{}
var _ resource.ResourceWithModifyPlan = &ActionResource{}
var _ resource.ResourceWithUpgradeState = &ActionResource{}

func (r *ActionResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if v, ok := request.ProviderData.(*clients.Client); ok {
		r.ProviderData = v
	}
}

func (r *ActionResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_resource_action"
}

func (r *ActionResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: migration.AzapiResourceActionMigrationV0ToV2(ctx),
		1: migration.AzapiResourceActionMigrationV1ToV2(ctx),
	}
}

func (r *ActionResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: docstrings.ID(),
			},

			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					myvalidator.StringIsResourceType(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: docstrings.Type(),
			},

			"resource_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					myvalidator.StringIsResourceID(),
				},
				MarkdownDescription: "The ID of an existing Azure source.",
			},

			"action": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: docstrings.ResourceAction(),
			},

			"method": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  defaults.StringDefault("POST"),
				Validators: []validator.String{
					stringvalidator.OneOf("POST", "PATCH", "PUT", "DELETE", "GET", "HEAD"),
				},
				MarkdownDescription: "Specifies the HTTP method of the azure resource action. Allowed values are `POST`, `PATCH`, `PUT` and `DELETE`. Defaults to `POST`.",
			},

			// The body attribute is a dynamic attribute that only allows users to specify the resource body as an HCL object
			"body": schema.DynamicAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Dynamic{
					myplanmodifier.DynamicUseStateWhen(dynamic.SemanticallyEqual),
				},
				MarkdownDescription: docstrings.Body(),
				Validators: []validator.Dynamic{
					myvalidator.DynamicIsNotStringValidator(),
				},
			},

			"when": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  defaults.StringDefault("apply"),
				Validators: []validator.String{
					stringvalidator.OneOf("apply", "destroy"),
				},
				MarkdownDescription: "When to perform the action, value must be one of: `apply`, `destroy`. Default is `apply`.",
			},

			"locks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(myvalidator.StringIsNotEmpty()),
				},
				MarkdownDescription: docstrings.Locks(),
			},

			"response_export_values": schema.DynamicAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Dynamic{
					myplanmodifier.DynamicUseStateWhen(dynamic.SemanticallyEqual),
				},
				MarkdownDescription: docstrings.ResponseExportValues(),
			},

			"sensitive_response_export_values": schema.DynamicAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Dynamic{
					myplanmodifier.DynamicUseStateWhen(dynamic.SemanticallyEqual),
				},
				MarkdownDescription: docstrings.ResponseExportValues(),
			},

			"output": schema.DynamicAttribute{
				Computed:            true,
				MarkdownDescription: docstrings.Output("azapi_resource_action"),
			},

			"sensitive_output": schema.DynamicAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: docstrings.SensitiveOutput("azapi_resource_action"),
			},

			"retry": retry.RetrySchema(ctx),

			"headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A map of headers to include in the request",
			},

			"query_parameters": schema.MapAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Optional:            true,
				MarkdownDescription: "A map of query parameters to include in the request",
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Read:   true,
				Delete: true,
			}),
		},

		Version: 2,
	}
}

func (r *ActionResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	var config, plan, state *ActionResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// destroy doesn't need to modify plan
	if config == nil {
		return
	}

	if state == nil || !dynamic.SemanticallyEqual(config.Body, state.Body) {
		plan.Output = basetypes.NewDynamicUnknown()
		plan.SensitiveOutput = basetypes.NewDynamicUnknown()
	} else {
		plan.Output = state.Output
		if !plan.ResponseExportValues.Equal(state.ResponseExportValues) {
			plan.Output = basetypes.NewDynamicUnknown()
		}
		plan.SensitiveOutput = state.SensitiveOutput
		if !plan.SensitiveResponseExportValues.Equal(state.SensitiveResponseExportValues) {
			plan.SensitiveOutput = basetypes.NewDynamicUnknown()
		}
	}

	response.Diagnostics.Append(response.Plan.Set(ctx, plan)...)
}

func (r *ActionResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var model ActionResourceModel
	if response.Diagnostics.Append(request.Plan.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	timeout, diags := model.Timeouts.Create(ctx, 30*time.Minute)
	if response.Diagnostics.Append(diags...); response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if model.When.ValueString() == "apply" {
		r.Action(ctx, model, &response.State, &response.Diagnostics)
	} else {
		id, err := parse.ResourceIDWithResourceType(model.ResourceId.ValueString(), model.Type.ValueString())
		if err != nil {
			response.Diagnostics.AddError("Invalid configuration", err.Error())
			return
		}
		resourceId := id.ID()
		if actionName := model.Action.ValueString(); actionName != "" {
			resourceId = fmt.Sprintf("%s/%s", id.ID(), actionName)
		}
		model.ID = basetypes.NewStringValue(resourceId)
		model.Output = basetypes.NewDynamicNull()
		model.SensitiveOutput = basetypes.NewDynamicNull()
		response.Diagnostics.Append(response.State.Set(ctx, model)...)
	}
}

func (r *ActionResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state, plan ActionResourceModel
	if response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...); response.Diagnostics.HasError() {
		return
	}
	if response.Diagnostics.Append(request.State.Get(ctx, &state)...); response.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, 30*time.Minute)
	if response.Diagnostics.Append(diags...); response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// See if we can skip the external API call (changes are to state only)
	if skip.CanSkipExternalRequest(state, plan, "update") {
		tflog.Debug(ctx, "azapi_resource.CreateUpdate skipping external request as no unskippable changes were detected")
		response.Diagnostics.Append(response.State.Set(ctx, plan)...)
		return
	}
	tflog.Debug(ctx, "azapi_resource.CreateUpdate proceeding with external request as no skippable changes were detected")

	if plan.When.ValueString() == "apply" {
		r.Action(ctx, plan, &response.State, &response.Diagnostics)
	}
}

func (r *ActionResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var model ActionResourceModel
	if response.Diagnostics.Append(request.State.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	if model.When.ValueString() == "destroy" {
		r.Action(ctx, model, &response.State, &response.Diagnostics)
	}
}

func (r *ActionResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state ActionResourceModel
	if response.Diagnostics.Append(request.State.Get(ctx, &state)...); response.Diagnostics.HasError() {
		return
	}

	if state.When.IsNull() {
		state.When = basetypes.NewStringValue("apply")
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (r *ActionResource) Action(ctx context.Context, model ActionResourceModel, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	actionTimeout, diags := model.Timeouts.Create(ctx, 30*time.Minute)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	id, err := parse.ResourceIDWithResourceType(model.ResourceId.ValueString(), model.Type.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid configuration", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	var requestBody interface{}
	if err := unmarshalBody(model.Body, &requestBody); err != nil {
		diagnostics.AddError("Invalid body", fmt.Sprintf(`The argument "body" is invalid: %s`, err.Error()))
		return
	}

	lockIds := AsStringList(model.Locks)
	slices.Sort(lockIds)
	for _, lockId := range lockIds {
		locks.ByID(lockId)
		defer locks.UnlockByID(lockId)
	}

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.ResourceClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	responseBody, err := client.Action(ctx, id.AzureResourceId, model.Action.ValueString(), id.ApiVersion, model.Method.ValueString(), requestBody, clients.NewRequestOptions(AsMapOfString(model.Headers), AsMapOfLists(model.QueryParameters)))
	if err != nil {
		diagnostics.AddError("Failed to perform action", fmt.Errorf("performing action %s of %q: %+v", model.Action.ValueString(), id, err).Error())
		return
	}

	resourceId := id.ID()
	if actionName := model.Action.ValueString(); actionName != "" {
		resourceId = fmt.Sprintf("%s/%s", id.ID(), actionName)
	}
	model.ID = basetypes.NewStringValue(resourceId)

	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, nil)
	if err != nil {
		diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	model.Output = output

	sensitiveOutput, err := buildOutputFromBody(responseBody, model.SensitiveResponseExportValues, nil)
	if err != nil {
		diagnostics.AddError("Failed to build sensitive output", err.Error())
		return
	}
	model.SensitiveOutput = sensitiveOutput

	diagnostics.Append(state.Set(ctx, model)...)
}
