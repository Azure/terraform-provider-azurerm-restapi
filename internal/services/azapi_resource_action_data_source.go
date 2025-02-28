package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/terraform-provider-azapi/internal/clients"
	"github.com/Azure/terraform-provider-azapi/internal/docstrings"
	"github.com/Azure/terraform-provider-azapi/internal/retry"
	"github.com/Azure/terraform-provider-azapi/internal/services/myvalidator"
	"github.com/Azure/terraform-provider-azapi/internal/services/parse"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ResourceActionDataSourceModel struct {
	ID                            types.String     `tfsdk:"id"`
	ResourceID                    types.String     `tfsdk:"resource_id"`
	Type                          types.String     `tfsdk:"type"`
	Action                        types.String     `tfsdk:"action"`
	Method                        types.String     `tfsdk:"method"`
	Body                          types.Dynamic    `tfsdk:"body"`
	ResponseExportValues          types.Dynamic    `tfsdk:"response_export_values"`
	SensitiveResponseExportValues types.Dynamic    `tfsdk:"sensitive_response_export_values"`
	Output                        types.Dynamic    `tfsdk:"output"`
	SensitiveOutput               types.Dynamic    `tfsdk:"sensitive_output"`
	Timeouts                      timeouts.Value   `tfsdk:"timeouts"`
	Retry                         retry.RetryValue `tfsdk:"retry"`
	Headers                       types.Map        `tfsdk:"headers"`
	QueryParameters               types.Map        `tfsdk:"query_parameters"`
}

type ResourceActionDataSource struct {
	ProviderData *clients.Client
}

var _ datasource.DataSource = &ResourceActionDataSource{}
var _ datasource.DataSourceWithConfigure = &ResourceActionDataSource{}

func (r *ResourceActionDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if v, ok := request.ProviderData.(*clients.Client); ok {
		r.ProviderData = v
	}
}

func (r *ResourceActionDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_resource_action"
}

func (r *ResourceActionDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: docstrings.ID(),
			},

			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					myvalidator.StringIsResourceType(),
				},
				MarkdownDescription: docstrings.Type(),
			},

			"resource_id": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					myvalidator.StringIsResourceID(),
				},
				MarkdownDescription: "The ID of the Azure resource to perform the action on.",
			},

			"action": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: docstrings.ResourceAction(),
			},

			"method": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("POST", "GET"),
				},
				MarkdownDescription: "The HTTP method to use when performing the action. Must be one of `POST`, `GET`. Defaults to `POST`.",
			},

			// The body attribute is a dynamic attribute that only allows users to specify the resource body as an HCL object
			"body": schema.DynamicAttribute{
				Optional: true,
				Validators: []validator.Dynamic{
					myvalidator.DynamicIsNotStringValidator(),
				},
			},

			"response_export_values": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: docstrings.ResponseExportValues(),
			},

			"sensitive_response_export_values": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: docstrings.SensitiveResponseExportValues(),
			},

			"output": schema.DynamicAttribute{
				Computed:            true,
				MarkdownDescription: docstrings.Output("data.azapi_resource_action"),
			},

			"sensitive_output": schema.DynamicAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: docstrings.SensitiveOutput("data.azapi_resource_action"),
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
				Read: true,
			}),
		},
	}
}

func (r *ResourceActionDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var model ResourceActionDataSourceModel
	if response.Diagnostics.Append(request.Config.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := model.Timeouts.Read(ctx, 5*time.Minute)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := parse.ResourceIDWithResourceType(model.ResourceID.ValueString(), model.Type.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid configuration", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "resource_id", id.ID())

	var requestBody interface{}
	if err := unmarshalBody(model.Body, &requestBody); err != nil {
		response.Diagnostics.AddError("Invalid body", fmt.Sprintf(`The argument "body" is invalid: %s`, err.Error()))
		return
	}

	method := model.Method.ValueString()
	if method == "" {
		method = "POST"
	}

	// Ensure the context deadline has been set before calling ConfigureClientWithCustomRetry().
	client := r.ProviderData.ResourceClient.ConfigureClientWithCustomRetry(ctx, model.Retry, false)

	responseBody, err := client.Action(ctx, id.AzureResourceId, model.Action.ValueString(), id.ApiVersion, method, requestBody, clients.NewRequestOptions(AsMapOfString(model.Headers), AsMapOfLists(model.QueryParameters)))
	if err != nil {
		response.Diagnostics.AddError("Failed to perform action", fmt.Errorf("performing action %s of %q: %+v", model.Action.ValueString(), id, err).Error())
		return
	}

	model.ID = basetypes.NewStringValue(id.ID())

	output, err := buildOutputFromBody(responseBody, model.ResponseExportValues, nil)
	if err != nil {
		response.Diagnostics.AddError("Failed to build output", err.Error())
		return
	}
	model.Output = output

	sensitiveOutput, err := buildOutputFromBody(responseBody, model.SensitiveResponseExportValues, nil)
	if err != nil {
		response.Diagnostics.AddError("Failed to build sensitive output", err.Error())
		return
	}
	model.SensitiveOutput = sensitiveOutput

	response.Diagnostics.Append(response.State.Set(ctx, &model)...)
}
