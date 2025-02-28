package services

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	"github.com/Azure/terraform-provider-azapi/internal/services/myplanmodifier/planmodifierdynamic"
	"github.com/Azure/terraform-provider-azapi/internal/services/myvalidator"
	"github.com/Azure/terraform-provider-azapi/internal/services/parse"
	"github.com/Azure/terraform-provider-azapi/internal/skip"
	"github.com/Azure/terraform-provider-azapi/internal/tf"
	"github.com/Azure/terraform-provider-azapi/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

type DataPlaneResourceModel struct {
	ID                            types.String     `tfsdk:"id"`
	Name                          types.String     `tfsdk:"name"`
	ParentID                      types.String     `tfsdk:"parent_id"`
	Type                          types.String     `tfsdk:"type"`
	Body                          types.Dynamic    `tfsdk:"body"`
	IgnoreCasing                  types.Bool       `tfsdk:"ignore_casing"`
	IgnoreMissingProperty         types.Bool       `tfsdk:"ignore_missing_property"`
	ReplaceTriggersExternalValues types.Dynamic    `tfsdk:"replace_triggers_external_values"`
	ReplaceTriggersRefs           types.List       `tfsdk:"replace_triggers_refs"`
	ResponseExportValues          types.Dynamic    `tfsdk:"response_export_values"`
	Retry                         retry.RetryValue `tfsdk:"retry" skip_on:"update"`
	Locks                         types.List       `tfsdk:"locks"`
	Output                        types.Dynamic    `tfsdk:"output"`
	Timeouts                      timeouts.Value   `tfsdk:"timeouts" skip_on:"update"`
	CreateHeaders                 types.Map        `tfsdk:"create_headers" skip_on:"update"`
	CreateQueryParameters         types.Map        `tfsdk:"create_query_parameters" skip_on:"update"`
	UpdateHeaders                 types.Map        `tfsdk:"update_headers"`
	UpdateQueryParameters         types.Map        `tfsdk:"update_query_parameters"`
	DeleteHeaders                 types.Map        `tfsdk:"delete_headers" skip_on:"update"`
	DeleteQueryParameters         types.Map        `tfsdk:"delete_query_parameters" skip_on:"update"`
	ReadHeaders                   types.Map        `tfsdk:"read_headers" skip_on:"update"`
	ReadQueryParameters           types.Map        `tfsdk:"read_query_parameters" skip_on:"update"`
}

type DataPlaneResource struct {
	ProviderData *clients.Client
}

var _ resource.Resource = &DataPlaneResource{}
var _ resource.ResourceWithConfigure = &DataPlaneResource{}
var _ resource.ResourceWithModifyPlan = &DataPlaneResource{}
var _ resource.ResourceWithUpgradeState = &DataPlaneResource{}

func (r *DataPlaneResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring azapi_data_plane_resource")
	if v, ok := request.ProviderData.(*clients.Client); ok {
		r.ProviderData = v
	}
}

func (r *DataPlaneResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_data_plane_resource"
}

func (r *DataPlaneResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: migration.AzapiDataPlaneResourceMigrationV0ToV2(ctx),
		1: migration.AzapiDataPlaneResourceMigrationV1ToV2(ctx),
	}
}

func (r *DataPlaneResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "This resource can manage some Azure data plane resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: docstrings.ID(),
			},

			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Specifies the name of the Azure resource. Changing this forces a new resource to be created.",
			},

			"parent_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					myvalidator.StringIsNotEmpty(),
				},
				MarkdownDescription: "The ID of the azure resource in which this resource is created. Changing this forces a new resource to be created.",
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
				Computed: true,
				// in the previous version, the default value is string "{}", now it's a dynamic value {}
				Default: defaults.DynamicDefault(types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})),
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
				MarkdownDescription: docstrings.Body(),
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

			"retry": retry.RetrySchema(ctx),

			"replace_triggers_external_values": schema.DynamicAttribute{
				Optional: true,
				MarkdownDescription: "Will trigger a replace of the resource when the value changes and is not `null`. This can be used by practitioners to force a replace of the resource when certain values change, e.g. changing the SKU of a virtual machine based on the value of variables or locals. " +
					"The value is a `dynamic`, so practitioners can compose the input however they wish. For a \"break glass\" set the value to `null` to prevent the plan modifier taking effect. \n" +
					"If you have `null` values that you do want to be tracked as affecting the resource replacement, include these inside an object. \n" +
					"Advanced use cases are possible and resource replacement can be triggered by values external to the resource, for example when a dependent resource changes.\n\n" +
					"e.g. to replace a resource when either the SKU or os_type attributes change:\n" +
					"\n" +
					"```hcl\n" +
					"resource \"azapi_data_plane_resource\" \"example\" {\n" +
					"  name = var.name\n" +
					"  type = \"Microsoft.AppConfiguration/configurationStores/keyValues@1.0\"\n" +
					"  body = {\n" +
					"    properties = {\n" +
					"      sku   = var.sku\n" +
					"      zones = var.zones\n" +
					"    }\n" +
					"  }\n" +
					"\n" +
					"  replace_triggers_external_values = [\n" +
					"    var.sku,\n" +
					"    var.zones,\n" +
					"  ]\n" +
					"}\n" +
					"```\n",
				PlanModifiers: []planmodifier.Dynamic{
					planmodifierdynamic.RequiresReplaceIfNotNull(),
				},
			},

			"replace_triggers_refs": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of paths in the current Terraform configuration. When the values at these paths change, the resource will be replaced.",
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
				MarkdownDescription: docstrings.Output("azapi_data_plane_resource"),
			},

			"create_headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A mapping of headers to be sent with the create request.",
			},

			"create_query_parameters": schema.MapAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Optional:            true,
				MarkdownDescription: "A mapping of query parameters to be sent with the create request.",
			},

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

			"delete_headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A mapping of headers to be sent with the delete request.",
			},

			"delete_query_parameters": schema.MapAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				Optional:            true,
				MarkdownDescription: "A mapping of query parameters to be sent with the delete request.",
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

func (r *DataPlaneResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	var config, plan, state *DataPlaneResourceModel
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

	if state == nil || !plan.ResponseExportValues.Equal(state.ResponseExportValues) || !dynamic.SemanticallyEqual(plan.Body, state.Body) {
		plan.Output = basetypes.NewDynamicUnknown()
	} else {
		plan.Output = state.Output
	}

	response.Diagnostics.Append(response.Plan.Set(ctx, plan)...)

	// Check if any paths in replace_triggers_refs have changed
	if state != nil && plan != nil && !plan.ReplaceTriggersRefs.IsNull() {
		refPaths := make(map[string]string)
		for pathIndex, refPath := range AsStringList(plan.ReplaceTriggersRefs) {
			refPaths[fmt.Sprintf("%d", pathIndex)] = refPath
		}

		// read previous values from state
		stateData, err := dynamic.ToJSON(state.Body)
		if err != nil {
			response.Diagnostics.AddError("Invalid state body configuration", err.Error())
			return
		}
		var stateModel interface{}
		err = json.Unmarshal(stateData, &stateModel)
		if err != nil {
			response.Diagnostics.AddError("Invalid state body configuration", err.Error())
			return
		}
		previousValues := flattenOutputJMES(stateModel, refPaths)

		// read current values from plan
		planData, err := dynamic.ToJSON(plan.Body)
		if err != nil {
			response.Diagnostics.AddError("Invalid plan body configuration", err.Error())
			return
		}
		var planModel interface{}
		err = json.Unmarshal(planData, &planModel)
		if err != nil {
			response.Diagnostics.AddError("Invalid plan body configuration", err.Error())
			return
		}
		currentValues := flattenOutputJMES(planModel, refPaths)

		// compare previous and current values
		if !reflect.DeepEqual(previousValues, currentValues) {
			response.RequiresReplace.Append(path.Root("body"))
		}
	}
}

func (r *DataPlaneResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	r.CreateUpdate(ctx, request.Plan, &response.State, &response.Diagnostics)
}

func (r *DataPlaneResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// See if we can skip the external API call (changes are to state only)
	var state, plan DataPlaneResourceModel
	if response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...); response.Diagnostics.HasError() {
		return
	}
	if response.Diagnostics.Append(request.State.Get(ctx, &state)...); response.Diagnostics.HasError() {
		return
	}
	if skip.CanSkipExternalRequest(state, plan, "update") {
		tflog.Debug(ctx, "azapi_resource.CreateUpdate skipping external request as no unskippable changes were detected")
		response.Diagnostics.Append(response.State.Set(ctx, plan)...)
	}
	tflog.Debug(ctx, "azapi_resource.CreateUpdate proceeding with external request as no skippable changes were detected")
	r.CreateUpdate(ctx, request.Plan, &response.State, &response.Diagnostics)
}

func (r *DataPlaneResource) CreateUpdate(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	var model DataPlaneResourceModel
	if diagnostics.Append(plan.Get(ctx, &model)...); diagnostics.HasError() {
		return
	}

	id, err := parse.NewDataPlaneResourceId(model.Name.ValueString(), model.ParentID.ValueString(), model.Type.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid configuration", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

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

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.DataPlaneClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	if isNewResource {
		// check if the resource already exists using the non-retry client to avoid issue where user specifies
		// a FooResourceNotFound error as a retryable error
		_, err = r.ProviderData.DataPlaneClient.Get(ctx, id, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
		if err == nil {
			diagnostics.AddError("Resource already exists", tf.ImportAsExistsError("azapi_data_plane_resource", id.ID()).Error())
			return
		}
		if !utils.ResponseErrorWasNotFound(err) {
			diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("checking for presence of existing %s: %+v", id, err).Error())
			return
		}
	}

	body := make(map[string]interface{})
	if err := unmarshalBody(model.Body, &body); err != nil {
		diagnostics.AddError("Invalid body", fmt.Sprintf(`The argument "body" is invalid: %s`, err.Error()))
		return
	}
	lockIds := AsStringList(model.Locks)
	slices.Sort(lockIds)
	for _, lockId := range lockIds {
		locks.ByID(lockId)
		defer locks.UnlockByID(lockId)
	}

	_, err = client.CreateOrUpdateThenPoll(ctx, id, body, clients.NewRequestOptions(AsMapOfString(model.CreateHeaders), AsMapOfLists(model.CreateQueryParameters)))
	if err != nil {
		diagnostics.AddError("Failed to create/update resource", fmt.Errorf("creating/updating %q: %+v", id, err).Error())
		return
	}

	// Create a new retry client to handle specific case of transient 403/404 after resource creation
	// If a read after create retry is not specified, use the default.
	clientGetAfterPut := r.ProviderData.DataPlaneClient.ConfigureClientWithCustomRetry(ctx, model.Retry, true)

	responseBody, err := clientGetAfterPut.Get(ctx, id, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
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

	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, nil)
	if err != nil {
		diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	model.Output = output

	diagnostics.Append(state.Set(ctx, model)...)
}

func (r *DataPlaneResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var model *DataPlaneResourceModel
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

	id, err := parse.DataPlaneResourceIDWithResourceType(model.ID.ValueString(), model.Type.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}
	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	// Ensure that the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.DataPlaneClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	responseBody, err := client.Get(ctx, id, clients.NewRequestOptions(AsMapOfString(model.ReadHeaders), AsMapOfLists(model.ReadQueryParameters)))
	if err != nil {
		if utils.ResponseErrorWasNotFound(err) {
			tflog.Info(ctx, fmt.Sprintf("[INFO] Error reading %q - removing from state", id.ID()))
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Failed to retrieve resource", fmt.Errorf("reading %s: %+v", id, err).Error())
		return
	}

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

	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, nil)
	if err != nil {
		response.Diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	model.Output = output

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
		model.Body = payload
	}

	model.Name = basetypes.NewStringValue(id.Name)
	model.ParentID = basetypes.NewStringValue(id.ParentId)
	model.Type = basetypes.NewStringValue(fmt.Sprintf("%s@%s", id.AzureResourceType, id.ApiVersion))

	response.Diagnostics.Append(response.State.Set(ctx, model)...)
}

func (r *DataPlaneResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var model *DataPlaneResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &model)...)
	if response.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := model.Timeouts.Delete(ctx, 30*time.Minute)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, err := parse.DataPlaneResourceIDWithResourceType(model.ID.ValueString(), model.Type.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.DataPlaneClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	lockIds := AsStringList(model.Locks)
	slices.Sort(lockIds)
	for _, lockId := range lockIds {
		locks.ByID(lockId)
		defer locks.UnlockByID(lockId)
	}

	_, err = client.DeleteThenPoll(ctx, id, clients.NewRequestOptions(AsMapOfString(model.DeleteHeaders), AsMapOfLists(model.DeleteQueryParameters)))
	if err != nil && !utils.ResponseErrorWasNotFound(err) {
		response.Diagnostics.AddError("Failed to delete resource", fmt.Errorf("deleting %s: %+v", id, err).Error())
	}
}
