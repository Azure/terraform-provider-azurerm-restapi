package services

import (
	"context"
	"encoding/json"
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
	"github.com/Azure/terraform-provider-azapi/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

type AzapiUpdateResourceModel struct {
	ID                    types.String     `tfsdk:"id"`
	Name                  types.String     `tfsdk:"name"`
	ParentID              types.String     `tfsdk:"parent_id"`
	ResourceID            types.String     `tfsdk:"resource_id"`
	Type                  types.String     `tfsdk:"type"`
	Body                  types.Dynamic    `tfsdk:"body"`
	IgnoreCasing          types.Bool       `tfsdk:"ignore_casing"`
	IgnoreMissingProperty types.Bool       `tfsdk:"ignore_missing_property"`
	ResponseExportValues  types.Dynamic    `tfsdk:"response_export_values"`
	Locks                 types.List       `tfsdk:"locks"`
	Output                types.Dynamic    `tfsdk:"output"`
	Timeouts              timeouts.Value   `tfsdk:"timeouts" skip_on:"update"`
	Retry                 retry.RetryValue `tfsdk:"retry" skip_on:"update"`
	UpdateHeaders         types.Map        `tfsdk:"update_headers"`
	UpdateQueryParameters types.Map        `tfsdk:"update_query_parameters"`
	ReadHeaders           types.Map        `tfsdk:"read_headers" skip_on:"update"`
	ReadQueryParameters   types.Map        `tfsdk:"read_query_parameters" skip_on:"update"`
}

type AzapiUpdateResource struct {
	ProviderData *clients.Client
}

var _ resource.Resource = &AzapiUpdateResource{}
var _ resource.ResourceWithConfigure = &AzapiUpdateResource{}
var _ resource.ResourceWithValidateConfig = &AzapiUpdateResource{}
var _ resource.ResourceWithModifyPlan = &AzapiUpdateResource{}
var _ resource.ResourceWithUpgradeState = &AzapiUpdateResource{}

func (r *AzapiUpdateResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if v, ok := request.ProviderData.(*clients.Client); ok {
		r.ProviderData = v
	}
}

func (r *AzapiUpdateResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_update_resource"
}

func (r *AzapiUpdateResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: migration.AzapiUpdateResourceMigrationV0ToV2(ctx),
		1: migration.AzapiUpdateResourceMigrationV1ToV2(ctx),
	}
}

func (r *AzapiUpdateResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "This resource can manage a subset of any existing Azure resource manager resource's properties.\n\n" +
			"-> **Note** This resource is used to add or modify properties on an existing resource. When delete `azapi_update_resource`, no operation will be performed, and these properties will stay unchanged. If you want to restore the modified properties to some values, you must apply the restored properties before deleting.",
		Description: "This resource can manage a subset of any existing Azure resource manager resource's properties.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: docstrings.ID(),
			},

			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					myvalidator.StringIsNotEmpty(),
				},
				MarkdownDescription: "Specifies the name of the Azure resource. Changing this forces a new resource to be created.",
			},

			"parent_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					myvalidator.StringIsResourceID(),
				},
				MarkdownDescription: docstrings.ParentID(),
			},

			"resource_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					myvalidator.StringIsResourceID(),
				},
				MarkdownDescription: "The ID of an existing Azure source.",
			},

			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					myvalidator.StringIsResourceType(),
				},
				MarkdownDescription: docstrings.Type(),
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

			"ignore_casing": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             defaults.BoolDefault(false),
				MarkdownDescription: docstrings.IgnoreCasing(),
			},

			"ignore_missing_property": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             defaults.BoolDefault(true),
				MarkdownDescription: docstrings.IgnoreMissingProperty(),
			},

			"response_export_values": schema.DynamicAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Dynamic{
					myplanmodifier.DynamicUseStateWhen(dynamic.SemanticallyEqual),
				},
				MarkdownDescription: docstrings.ResponseExportValues(),
			},

			"locks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(myvalidator.StringIsNotEmpty()),
				},
				MarkdownDescription: docstrings.Locks(),
			},

			"output": schema.DynamicAttribute{
				Computed:            true,
				MarkdownDescription: docstrings.Output("azapi_update_resource"),
			},

			"retry": retry.RetrySchema(ctx),

			"update_headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A mapping of headers to be sent with the update request.",
			},

			"update_query_parameters": schema.MapAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Optional:            true,
				MarkdownDescription: "A mapping of query parameters to be sent with the update request.",
			},

			"read_headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A mapping of headers to be sent with the read request.",
			},

			"read_query_parameters": schema.MapAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Optional:            true,
				MarkdownDescription: "A mapping of query parameters to be sent with the read request.",
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

func (r *AzapiUpdateResource) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config *AzapiUpdateResourceModel
	if response.Diagnostics.Append(request.Config.Get(ctx, &config)...); response.Diagnostics.HasError() {
		return
	}

	if config == nil {
		return
	}

	if config.Name.IsNull() && !config.ParentID.IsNull() {
		response.Diagnostics.AddError("Invalid configuration", `The argument "name" is required when the argument "parent_id" is set`)
	}
	if !config.Name.IsNull() && config.ParentID.IsNull() {
		response.Diagnostics.AddError("Invalid configuration", `The argument "parent_id" is required when the argument "name" is set`)
	}
	if config.Name.IsNull() && config.ResourceID.IsNull() {
		response.Diagnostics.AddError("Invalid configuration", `One of the arguments "name" or "resource_id" must be set`)
	}
	if !config.Name.IsNull() && !config.ResourceID.IsNull() {
		response.Diagnostics.AddError("Invalid configuration", `Only one of the arguments "name" or "resource_id" can be set`)
	}
	if response.Diagnostics.HasError() {
		return
	}

	if name := config.Name.ValueString(); name != "" {
		parentId := config.ParentID.ValueString()
		resourceType := config.Type.ValueString()

		// verify parent_id when it's known
		if parentId != "" {
			_, err := parse.NewResourceID(name, parentId, resourceType)
			if err != nil {
				response.Diagnostics.AddError("Invalid configuration", err.Error())
				return
			}
		}
	}
}

func (r *AzapiUpdateResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	var config, state, plan *AzapiUpdateResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	// destroy doesn't need to modify plan
	if config == nil {
		return
	}

	if state == nil || !plan.ResponseExportValues.Equal(state.ResponseExportValues) || !dynamic.SemanticallyEqual(plan.Body, state.Body) || !plan.Type.Equal(state.Type) {
		plan.Output = basetypes.NewDynamicUnknown()
	} else {
		plan.Output = state.Output
	}

	response.Diagnostics.Append(response.Plan.Set(ctx, plan)...)
}

func (r *AzapiUpdateResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	r.CreateUpdate(ctx, request.Plan, &response.State, &response.Diagnostics)
}

func (r *AzapiUpdateResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// See if we can skip the external API call (changes are to state only)
	var plan, state AzapiUpdateResourceModel
	if response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...); response.Diagnostics.HasError() {
		return
	}
	if response.Diagnostics.Append(request.State.Get(ctx, &state)...); response.Diagnostics.HasError() {
		return
	}
	if skip.CanSkipExternalRequest(plan, state, "update") {
		tflog.Debug(ctx, "azapi_resource.CreateUpdate skipping external request as no unskippable changes were detected")
		response.Diagnostics.Append(request.State.Set(ctx, plan)...)
		return
	}
	tflog.Debug(ctx, "azapi_resource.CreateUpdate proceeding with external request as no skippable changes were detected")

	r.CreateUpdate(ctx, request.Plan, &response.State, &response.Diagnostics)
}

func (r *AzapiUpdateResource) CreateUpdate(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	var model AzapiUpdateResourceModel
	if diagnostics.Append(plan.Get(ctx, &model)...); diagnostics.HasError() {
		return
	}

	isNewResource := state == nil || state.Raw.IsNull()

	var timeout time.Duration
	var diags diag.Diagnostics
	if isNewResource {
		timeout, diags = model.Timeouts.Create(ctx, 30*time.Minute)
		if diagnostics.Append(diags...); diagnostics.HasError() {
			return
		}
	} else {
		timeout, diags = model.Timeouts.Update(ctx, 30*time.Minute)
		if diagnostics.Append(diags...); diagnostics.HasError() {
			return
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var id parse.ResourceId
	// We need to ensure that the ID parsed in create and update is the same to produce consistent results.
	// In update, all these fields are set, using resource_id and type is able to parse the parent_id and name which are used to build it.
	// But using parent_id, name and type is not able to parse the original resource_id, because the last resource type segment comes from the type instead of the resource_id.
	if resourceId := model.ResourceID.ValueString(); len(resourceId) != 0 {
		buildId, err := parse.ResourceIDWithResourceType(model.ResourceID.ValueString(), model.Type.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid configuration", err.Error())
			return
		}
		id = buildId
	} else {
		buildId, err := parse.NewResourceID(model.Name.ValueString(), model.ParentID.ValueString(), model.Type.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid configuration", err.Error())
			return
		}
		id = buildId
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.ResourceClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	existing, err := client.Get(ctx, id.AzureResourceId, id.ApiVersion, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
	if err != nil {
		diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("checking for presence of existing %s: %+v", id, err).Error())
		return
	}
	if utils.GetId(existing) == nil {
		diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("update target does not exist %s", id).Error())
		return
	}

	var requestBody interface{}
	if err := unmarshalBody(model.Body, &requestBody); err != nil {
		diagnostics.AddError("Invalid body", fmt.Sprintf(`The argument "body" is invalid: err: %+v`, err))
		return
	}

	requestBody = utils.MergeObject(existing, requestBody)

	if id.ResourceDef != nil {
		requestBody = (*id.ResourceDef).GetWriteOnly(utils.NormalizeObject(requestBody))
	}

	lockIds := AsStringList(model.Locks)
	slices.Sort(lockIds)
	for _, lockId := range lockIds {
		locks.ByID(lockId)
		defer locks.UnlockByID(lockId)
	}

	_, err = client.CreateOrUpdate(ctx, id.AzureResourceId, id.ApiVersion, requestBody, clients.NewRequestOptions(AsMapOfString(model.UpdateHeaders), AsMapOfLists(model.UpdateQueryParameters)))
	if err != nil {
		diagnostics.AddError("Failed to update resource", fmt.Errorf("updating %q: %+v", id, err).Error())
		return
	}

	responseBody, err := client.Get(ctx, id.AzureResourceId, id.ApiVersion, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
	if err != nil {
		if utils.ResponseErrorWasNotFound(err) {
			tflog.Info(ctx, fmt.Sprintf("Error reading %q - removing from state", id.ID()))
			state.RemoveResource(ctx)
			return
		}
		diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("reading %s: %+v", id, err).Error())
		return
	}

	model.ID = basetypes.NewStringValue(id.ID())
	model.Name = basetypes.NewStringValue(id.Name)
	model.ParentID = basetypes.NewStringValue(id.ParentId)
	model.ResourceID = basetypes.NewStringValue(id.AzureResourceId)

	var defaultOutput interface{}
	if !r.ProviderData.Features.DisableDefaultOutput {
		defaultOutput = id.ResourceDef.GetReadOnly(responseBody)
		defaultOutput = utils.RemoveFields(defaultOutput, volatileFieldList())
	}
	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, defaultOutput)
	if err != nil {
		diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	model.Output = output

	diagnostics.Append(state.Set(ctx, model)...)
}

func (r *AzapiUpdateResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var model AzapiUpdateResourceModel
	if response.Diagnostics.Append(request.State.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, 5*time.Minute)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := parse.ResourceIDWithResourceType(model.ID.ValueString(), model.Type.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid resource id", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.ResourceClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	responseBody, err := client.Get(ctx, id.AzureResourceId, id.ApiVersion, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
	if err != nil {
		if utils.ResponseErrorWasNotFound(err) {
			tflog.Info(ctx, fmt.Sprintf("[INFO] Error reading %q - removing from state", id.ID()))
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("reading %q: %+v", id, err).Error())
		return
	}

	state := model
	state.ID = basetypes.NewStringValue(id.ID())
	state.Name = basetypes.NewStringValue(id.Name)
	state.ParentID = basetypes.NewStringValue(id.ParentId)
	state.ResourceID = basetypes.NewStringValue(id.AzureResourceId)
	state.Type = basetypes.NewStringValue(fmt.Sprintf("%s@%s", id.AzureResourceType, id.ApiVersion))

	requestBody := make(map[string]interface{})
	if err := unmarshalBody(model.Body, &requestBody); err != nil {
		response.Diagnostics.AddError("Invalid body", fmt.Sprintf(`The argument "body" is invalid: %s`, err.Error()))
		return
	}

	option := utils.UpdateJsonOption{
		IgnoreCasing:          model.IgnoreCasing.ValueBool(),
		IgnoreMissingProperty: model.IgnoreMissingProperty.ValueBool(),
	}
	body := utils.UpdateObject(requestBody, responseBody, option)

	data, err := json.Marshal(body)
	if err != nil {
		response.Diagnostics.AddError("Invalid body", err.Error())
		return
	}

	var defaultOutput interface{}
	if !r.ProviderData.Features.DisableDefaultOutput {
		defaultOutput = id.ResourceDef.GetReadOnly(responseBody)
		defaultOutput = utils.RemoveFields(defaultOutput, volatileFieldList())
	}
	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, defaultOutput)
	if err != nil {
		response.Diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	state.Output = output

	if !model.Body.IsNull() {
		payload, err := dynamic.FromJSON(data, model.Body.UnderlyingValue().Type(ctx))
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to parse payload: %s", err.Error()))
			payload, err = dynamic.FromJSONImplied(data)
			if err != nil {
				response.Diagnostics.AddError("Invalid payload", err.Error())
				return
			}
		}
		state.Body = payload
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (r *AzapiUpdateResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {

}
