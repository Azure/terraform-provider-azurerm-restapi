package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/terraform-provider-azapi/internal/azure"
	"github.com/Azure/terraform-provider-azapi/internal/azure/location"
	"github.com/Azure/terraform-provider-azapi/internal/azure/tags"
	"github.com/Azure/terraform-provider-azapi/internal/clients"
	"github.com/Azure/terraform-provider-azapi/internal/features"
	"github.com/Azure/terraform-provider-azapi/internal/services"
	"github.com/Azure/terraform-provider-azapi/internal/services/functions"
	"github.com/Azure/terraform-provider-azapi/internal/services/myvalidator"
	"github.com/Azure/terraform-provider-azapi/version"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ provider.Provider = &Provider{}
var _ provider.ProviderWithFunctions = &Provider{}
var _ provider.ProviderWithEphemeralResources = &Provider{}

func AzureProvider() provider.Provider {
	return &Provider{}
}

type Provider struct {
}

type providerData struct {
	SubscriptionID               types.String `tfsdk:"subscription_id"`
	ClientID                     types.String `tfsdk:"client_id"`
	ClientIDFilePath             types.String `tfsdk:"client_id_file_path"`
	TenantID                     types.String `tfsdk:"tenant_id"`
	AuxiliaryTenantIDs           types.List   `tfsdk:"auxiliary_tenant_ids"`
	Endpoint                     types.List   `tfsdk:"endpoint"`
	Environment                  types.String `tfsdk:"environment"`
	ClientCertificate            types.String `tfsdk:"client_certificate"`
	ClientCertificatePath        types.String `tfsdk:"client_certificate_path"`
	ClientCertificatePassword    types.String `tfsdk:"client_certificate_password"`
	ClientSecret                 types.String `tfsdk:"client_secret"`
	ClientSecretFilePath         types.String `tfsdk:"client_secret_file_path"`
	SkipProviderRegistration     types.Bool   `tfsdk:"skip_provider_registration"`
	OIDCRequestToken             types.String `tfsdk:"oidc_request_token"`
	OIDCRequestURL               types.String `tfsdk:"oidc_request_url"`
	OIDCToken                    types.String `tfsdk:"oidc_token"`
	OIDCTokenFilePath            types.String `tfsdk:"oidc_token_file_path"`
	OIDCAzureServiceConnectionID types.String `tfsdk:"oidc_azure_service_connection_id"`
	UseOIDC                      types.Bool   `tfsdk:"use_oidc"`
	UseCLI                       types.Bool   `tfsdk:"use_cli"`
	UseMSI                       types.Bool   `tfsdk:"use_msi"`
	UseAKSWorkloadIdentity       types.Bool   `tfsdk:"use_aks_workload_identity"`
	PartnerID                    types.String `tfsdk:"partner_id"`
	CustomCorrelationRequestID   types.String `tfsdk:"custom_correlation_request_id"`
	DisableCorrelationRequestID  types.Bool   `tfsdk:"disable_correlation_request_id"`
	DisableTerraformPartnerID    types.Bool   `tfsdk:"disable_terraform_partner_id"`
	DefaultName                  types.String `tfsdk:"default_name"`
	DefaultLocation              types.String `tfsdk:"default_location"`
	DefaultTags                  types.Map    `tfsdk:"default_tags"`
	EnablePreflight              types.Bool   `tfsdk:"enable_preflight"`
	DisableDefaultOutput         types.Bool   `tfsdk:"disable_default_output"`
}

func (model providerData) GetClientId() (*string, error) {
	clientId := strings.TrimSpace(model.ClientID.ValueString())

	if path := model.ClientIDFilePath.ValueString(); path != "" {
		// #nosec G304
		fileClientIdRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client ID from file %q: %v", path, err)
		}

		fileClientId := strings.TrimSpace(string(fileClientIdRaw))

		if clientId != "" && clientId != fileClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and supplied Client ID file contents - please either remove one or ensure they match")
		}

		clientId = fileClientId
	}

	if model.UseAKSWorkloadIdentity.ValueBool() && os.Getenv("AZURE_CLIENT_ID") != "" {
		aksClientId := os.Getenv("AZURE_CLIENT_ID")
		if clientId != "" && clientId != aksClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
		}
		clientId = aksClientId
	}

	return &clientId, nil
}

func (model providerData) GetClientSecret() (*string, error) {
	clientSecret := strings.TrimSpace(model.ClientSecret.ValueString())

	if path := model.ClientSecretFilePath.ValueString(); path != "" {
		// #nosec G304
		fileSecretRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client Secret from file %q: %v", path, err)
		}

		fileSecret := strings.TrimSpace(string(fileSecretRaw))

		if clientSecret != "" && clientSecret != fileSecret {
			return nil, fmt.Errorf("mismatch between supplied Client Secret and supplied Client Secret file contents - please either remove one or ensure they match")
		}

		clientSecret = fileSecret
	}

	return &clientSecret, nil
}

func (model providerData) GetOIDCTokenFilePath() string {
	if !model.OIDCTokenFilePath.IsNull() && model.OIDCTokenFilePath.ValueString() != "" {
		return model.OIDCTokenFilePath.ValueString()
	}

	if model.UseAKSWorkloadIdentity.ValueBool() && os.Getenv("AZURE_FEDERATED_TOKEN_FILE") != "" {
		return os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	}

	return ""
}

type providerEndpointData struct {
	ActiveDirectoryAuthorityHost types.String `tfsdk:"active_directory_authority_host"`
	ResourceManagerEndpoint      types.String `tfsdk:"resource_manager_endpoint"`
	ResourceManagerAudience      types.String `tfsdk:"resource_manager_audience"`
}

func (p Provider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "azapi"
}

func (p Provider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "The Azure API Provider",
		Attributes: map[string]schema.Attribute{
			"subscription_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Subscription ID which should be used. This can also be sourced from the `ARM_SUBSCRIPTION_ID` Environment Variable.",
			},

			"client_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Client ID which should be used. This can also be sourced from the `ARM_CLIENT_ID` Environment Variable.",
			},

			"client_id_file_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a file containing the Client ID which should be used. This can also be sourced from the `ARM_CLIENT_ID_FILE_PATH` Environment Variable.",
			},

			"tenant_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Tenant ID should be used. This can also be sourced from the `ARM_TENANT_ID` Environment Variable.",
			},

			"auxiliary_tenant_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          []validator.List{listvalidator.SizeAtMost(3)},
				MarkdownDescription: "List of auxiliary Tenant IDs required for multi-tenancy and cross-tenant scenarios. This can also be sourced from the `ARM_AUXILIARY_TENANT_IDS` Environment Variable.",
			},

			"endpoint": schema.ListNestedAttribute{
				Optional:            true,
				Validators:          []validator.List{listvalidator.SizeAtMost(1)},
				MarkdownDescription: "The Azure API Endpoint Configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"active_directory_authority_host": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The Azure Resource Manager endpoint to use. This can also be sourced from the `ARM_RESOURCE_MANAGER_ENDPOINT` Environment Variable. Defaults to `https://management.azure.com/` for public cloud.",
						},

						"resource_manager_endpoint": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The resource ID to obtain AD tokens for. This can also be sourced from the `ARM_RESOURCE_MANAGER_AUDIENCE` Environment Variable. Defaults to `https://management.core.windows.net/` for public cloud.",
						},

						"resource_manager_audience": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The Azure Active Directory login endpoint to use. This can also be sourced from the `ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST` Environment Variable. Defaults to `https://login.microsoftonline.com/` for public cloud.",
						},
					},
				},
			},

			"environment": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("public", "usgovernment", "china"),
				},
				MarkdownDescription: "The Cloud Environment which should be used. Possible values are `public`, `usgovernment` and `china`. Defaults to `public`. This can also be sourced from the `ARM_ENVIRONMENT` Environment Variable.",
			},

			// TODO@mgd: the metadata_host is used to retrieve metadata from Azure to identify current environment, this is used to eliminate Azure Stack usage, in which case the provider doesn't support.
			// "metadata_host": {
			// 	Type:        schema.TypeString,
			// 	Required:    true,
			// 	DefaultFunc: schema.EnvDefaultFunc("ARM_METADATA_HOSTNAME", ""),
			// 	Description: "The Hostname which should be used for the Azure Metadata Service.",
			// },

			// Client Certificate specific fields
			"client_certificate_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to the Client Certificate associated with the Service Principal which should be used. This can also be sourced from the `ARM_CLIENT_CERTIFICATE_PATH` Environment Variable.",
			},

			"client_certificate": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A base64-encoded PKCS#12 bundle to be used as the client certificate for authentication. This can also be sourced from the `ARM_CLIENT_CERTIFICATE` environment variable.",
			},

			"client_certificate_password": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The password associated with the Client Certificate. This can also be sourced from the `ARM_CLIENT_CERTIFICATE_PASSWORD` Environment Variable.",
			},

			// Client Secret specific fields
			"client_secret": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Client Secret which should be used. This can also be sourced from the `ARM_CLIENT_SECRET` Environment Variable.",
			},

			"client_secret_file_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a file containing the Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret. This can also be sourced from the `ARM_CLIENT_SECRET_FILE_PATH` Environment Variable.",
			},

			"skip_provider_registration": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Should the Provider skip registering the Resource Providers it supports? This can also be sourced from the `ARM_SKIP_PROVIDER_REGISTRATION` Environment Variable. Defaults to `false`.",
			},

			// OIDC specific fields
			"oidc_request_token": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The bearer token for the request to the OIDC provider. This can also be sourced from the `ARM_OIDC_REQUEST_TOKEN` or `ACTIONS_ID_TOKEN_REQUEST_TOKEN` Environment Variables.",
			},

			"oidc_request_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The URL for the OIDC provider from which to request an ID token. This can also be sourced from the `ARM_OIDC_REQUEST_URL` or `ACTIONS_ID_TOKEN_REQUEST_URL` Environment Variables.",
			},

			"oidc_token": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID token when authenticating using OpenID Connect (OIDC). This can also be sourced from the `ARM_OIDC_TOKEN` environment Variable.",
			},

			"oidc_token_file_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a file containing an ID token when authenticating using OpenID Connect (OIDC). This can also be sourced from the `ARM_OIDC_TOKEN_FILE_PATH` environment Variable.",
			},

			"oidc_azure_service_connection_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Azure Pipelines Service Connection ID to use for authentication. This can also be sourced from the `ARM_OIDC_AZURE_SERVICE_CONNECTION_ID` environment variable.",
			},

			"use_oidc": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Should OIDC be used for Authentication? This can also be sourced from the `ARM_USE_OIDC` Environment Variable. Defaults to `false`.",
			},

			// Azure CLI specific fields
			"use_cli": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Should Azure CLI be used for authentication? This can also be sourced from the `ARM_USE_CLI` environment variable. Defaults to `true`.",
			},

			// Managed Service Identity specific fields
			"use_msi": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Should Managed Identity be used for Authentication? This can also be sourced from the `ARM_USE_MSI` Environment Variable. Defaults to `false`.",
			},

			"use_aks_workload_identity": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Should AKS Workload Identity be used for Authentication? This can also be sourced from the `ARM_USE_AKS_WORKLOAD_IDENTITY` Environment Variable. Defaults to `false`. When set, `client_id`, `tenant_id` and `oidc_token_file_path` will be detected from the environment and do not need to be specified.",
			},

			// TODO@mgd: azidentity doesn't support msi_endpoint
			// "msi_endpoint": {
			// 	Type:        schema.TypeString,
			// 	Optional:    true,
			// 	DefaultFunc: schema.EnvDefaultFunc("ARM_MSI_ENDPOINT", ""),
			// 	Description: "The path to a custom endpoint for Managed Service Identity - in most circumstances this should be detected automatically. ",
			// },

			// Managed Tracking GUID for User-agent
			"partner_id": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.Any(myvalidator.StringIsUUID(), myvalidator.StringIsEmpty()),
				},
				MarkdownDescription: "A GUID/UUID that is [registered](https://docs.microsoft.com/azure/marketplace/azure-partner-customer-usage-attribution#register-guids-and-offers) with Microsoft to facilitate partner resource usage attribution. This can also be sourced from the `ARM_PARTNER_ID` Environment Variable.",
			},

			"custom_correlation_request_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The value of the `x-ms-correlation-request-id` header, otherwise an auto-generated UUID will be used. This can also be sourced from the `ARM_CORRELATION_REQUEST_ID` environment variable.",
			},

			"disable_correlation_request_id": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "This will disable the x-ms-correlation-request-id header.",
			},

			"disable_terraform_partner_id": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disable sending the Terraform Partner ID if a custom `partner_id` isn't specified, which allows Microsoft to better understand the usage of Terraform. The Partner ID does not give HashiCorp any direct access to usage information. This can also be sourced from the `ARM_DISABLE_TERRAFORM_PARTNER_ID` environment variable. Defaults to `false`.",
			},

			"default_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The default name to create the azure resource. The `name` in each resource block can override the `default_name`. Changing this forces new resources to be created.",
			},

			"default_location": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: " The default Azure Region where the azure resource should exist. The `location` in each resource block can override the `default_location`. Changing this forces new resources to be created.",
			},

			"default_tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					tags.Validator(),
				},
				MarkdownDescription: "A mapping of tags which should be assigned to the azure resource as default tags. The`tags` in each resource block can override the `default_tags`.",
			},

			"enable_preflight": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable Preflight Validation. The default is false. When set to true, the provider will use Preflight to do static validation before really deploying a new resource. When set to false, the provider will disable this validation.",
			},

			"disable_default_output": schema.BoolAttribute{
				Optional:    true,
				Description: "Disable default output. The default is false. When set to false, the provider will output the read-only properties if `response_export_values` is not specified in the resource block. When set to true, the provider will disable this output.",
			},
		},
	}
}

func (p Provider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var model providerData
	if response.Diagnostics.Append(request.Config.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	// set the defaults from environment variables
	if model.SubscriptionID.IsNull() {
		if v := os.Getenv("ARM_SUBSCRIPTION_ID"); v != "" {
			model.SubscriptionID = types.StringValue(v)
		}
	}
	if model.ClientID.IsNull() {
		if v := os.Getenv("ARM_CLIENT_ID"); v != "" {
			model.ClientID = types.StringValue(v)
		}
	}
	if model.ClientIDFilePath.IsNull() {
		if v := os.Getenv("ARM_CLIENT_ID_FILE_PATH"); v != "" {
			model.ClientIDFilePath = types.StringValue(v)
		}
	}

	if model.UseAKSWorkloadIdentity.IsNull() {
		if v := os.Getenv("ARM_USE_AKS_WORKLOAD_IDENTITY"); v != "" {
			model.UseAKSWorkloadIdentity = types.BoolValue(v == "true")
		} else {
			model.UseAKSWorkloadIdentity = types.BoolValue(false)
		}
	}

	if model.TenantID.IsNull() {
		if v := os.Getenv("ARM_TENANT_ID"); v != "" {
			model.TenantID = types.StringValue(v)
		}
		if model.UseAKSWorkloadIdentity.ValueBool() && os.Getenv("AZURE_TENANT_ID") != "" {
			aksTenantID := os.Getenv("AZURE_TENANT_ID")
			if model.TenantID.ValueString() != "" && model.TenantID.ValueString() != aksTenantID {
				response.Diagnostics.AddError("Invalid `tenant_id` value", "mismatch between supplied Tenant ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
				return
			}
			model.TenantID = types.StringValue(aksTenantID)
		}
	}

	if model.Endpoint.IsNull() {
		activeDirectoryAuthorityHost := os.Getenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
		resourceManagerEndpoint := os.Getenv("ARM_RESOURCE_MANAGER_ENDPOINT")
		resourceManagerAudience := os.Getenv("ARM_RESOURCE_MANAGER_AUDIENCE")
		attrTypes := make(map[string]attr.Type)
		attrTypes["active_directory_authority_host"] = types.StringType
		attrTypes["resource_manager_endpoint"] = types.StringType
		attrTypes["resource_manager_audience"] = types.StringType
		model.Endpoint = types.ListValueMust(types.ObjectType{
			AttrTypes: attrTypes,
		}, []attr.Value{
			types.ObjectValueMust(attrTypes, map[string]attr.Value{
				"active_directory_authority_host": types.StringValue(activeDirectoryAuthorityHost),
				"resource_manager_endpoint":       types.StringValue(resourceManagerEndpoint),
				"resource_manager_audience":       types.StringValue(resourceManagerAudience),
			}),
		})
	}

	if model.Environment.IsNull() {
		if v := os.Getenv("ARM_ENVIRONMENT"); v != "" {
			model.Environment = types.StringValue(v)
		} else {
			model.Environment = types.StringValue("public")
		}
	}

	if model.AuxiliaryTenantIDs.IsNull() {
		if v := os.Getenv("ARM_AUXILIARY_TENANT_IDS"); v != "" {
			values := make([]attr.Value, 0)
			for _, v := range strings.Split(v, ";") {
				values = append(values, types.StringValue(v))
			}
			model.AuxiliaryTenantIDs = types.ListValueMust(types.StringType, values)
		}
	}

	if model.ClientCertificate.IsNull() {
		if v := os.Getenv("ARM_CLIENT_CERTIFICATE"); v != "" {
			model.ClientCertificate = types.StringValue(v)
		}
	}

	if model.ClientCertificatePath.IsNull() {
		if v := os.Getenv("ARM_CLIENT_CERTIFICATE_PATH"); v != "" {
			model.ClientCertificatePath = types.StringValue(v)
		}
	}

	if model.ClientCertificatePassword.IsNull() {
		if v := os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD"); v != "" {
			model.ClientCertificatePassword = types.StringValue(v)
		}
	}

	if model.ClientSecret.IsNull() {
		if v := os.Getenv("ARM_CLIENT_SECRET"); v != "" {
			model.ClientSecret = types.StringValue(v)
		}
	}

	if model.ClientSecretFilePath.IsNull() {
		if v := os.Getenv("ARM_CLIENT_SECRET_FILE_PATH"); v != "" {
			model.ClientSecretFilePath = types.StringValue(v)
		}
	}

	if model.SkipProviderRegistration.IsNull() {
		if v := os.Getenv("ARM_SKIP_PROVIDER_REGISTRATION"); v != "" {
			model.SkipProviderRegistration = types.BoolValue(v == "true")
		} else {
			model.SkipProviderRegistration = types.BoolValue(false)
		}
	}

	if model.OIDCRequestToken.IsNull() {
		if v := os.Getenv("ARM_OIDC_REQUEST_TOKEN"); v != "" {
			model.OIDCRequestToken = types.StringValue(v)
		} else if v := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN"); v != "" {
			model.OIDCRequestToken = types.StringValue(v)
		}
	}

	if model.OIDCRequestURL.IsNull() {
		if v := os.Getenv("ARM_OIDC_REQUEST_URL"); v != "" {
			model.OIDCRequestURL = types.StringValue(v)
		} else if v := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"); v != "" {
			model.OIDCRequestURL = types.StringValue(v)
		}
	}

	if model.OIDCToken.IsNull() {
		if v := os.Getenv("ARM_OIDC_TOKEN"); v != "" {
			model.OIDCToken = types.StringValue(v)
		}
	}

	if model.OIDCTokenFilePath.IsNull() {
		if v := os.Getenv("ARM_OIDC_TOKEN_FILE_PATH"); v != "" {
			model.OIDCTokenFilePath = types.StringValue(v)
		}
	}

	if model.OIDCAzureServiceConnectionID.IsNull() {
		if v := os.Getenv("ARM_OIDC_AZURE_SERVICE_CONNECTION_ID"); v != "" {
			model.OIDCAzureServiceConnectionID = types.StringValue(v)
		}
	}

	if model.UseOIDC.IsNull() {
		if v := os.Getenv("ARM_USE_OIDC"); v != "" {
			model.UseOIDC = types.BoolValue(v == "true")
		} else {
			model.UseOIDC = types.BoolValue(false)
		}
	}

	if model.UseCLI.IsNull() {
		if v := os.Getenv("ARM_USE_CLI"); v != "" {
			model.UseCLI = types.BoolValue(v == "true")
		} else {
			model.UseCLI = types.BoolValue(true)
		}
	}

	if model.UseMSI.IsNull() {
		if v := os.Getenv("ARM_USE_MSI"); v != "" {
			model.UseMSI = types.BoolValue(v == "true")
		} else {
			model.UseMSI = types.BoolValue(false)
		}
	}

	if model.PartnerID.IsNull() {
		if v := os.Getenv("ARM_PARTNER_ID"); v != "" {
			model.PartnerID = types.StringValue(v)
		}
	}

	if model.CustomCorrelationRequestID.IsNull() {
		if v := os.Getenv("ARM_CORRELATION_REQUEST_ID"); v != "" {
			model.CustomCorrelationRequestID = types.StringValue(v)
		}
	}

	if model.DisableCorrelationRequestID.IsNull() {
		if v := os.Getenv("ARM_DISABLE_CORRELATION_REQUEST_ID"); v != "" {
			model.DisableCorrelationRequestID = types.BoolValue(v == "true")
		} else {
			model.DisableCorrelationRequestID = types.BoolValue(false)
		}
	}

	if model.DisableTerraformPartnerID.IsNull() {
		if v := os.Getenv("ARM_DISABLE_TERRAFORM_PARTNER_ID"); v != "" {
			model.DisableTerraformPartnerID = types.BoolValue(v == "true")
		} else {
			model.DisableTerraformPartnerID = types.BoolValue(false)
		}
	}

	if model.EnablePreflight.IsNull() {
		model.EnablePreflight = types.BoolValue(false)
	}
	if model.DisableDefaultOutput.IsNull() {
		model.DisableDefaultOutput = types.BoolValue(false)
	}

	var cloudConfig cloud.Configuration
	env := model.Environment.ValueString()
	switch strings.ToLower(env) {
	case "public":
		cloudConfig = cloud.AzurePublic
	case "usgovernment":
		cloudConfig = cloud.AzureGovernment
	case "china":
		cloudConfig = cloud.AzureChina
	default:
		response.Diagnostics.AddError("Invalid `environment` value.", fmt.Sprintf("The `environment` value '%s' is invalid. Valid values are 'public', 'usgovernment' and 'china'.", env))
		return
	}

	if elements := model.Endpoint.Elements(); len(elements) != 0 {
		var endpoint providerEndpointData
		diags := elements[0].(basetypes.ObjectValue).As(ctx, &endpoint, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		})
		response.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		resourceManagerEndpoint := cloudConfig.Services[cloud.ResourceManager].Endpoint
		resourceManagerAudience := cloudConfig.Services[cloud.ResourceManager].Audience
		if v := endpoint.ResourceManagerEndpoint.ValueString(); v != "" {
			resourceManagerEndpoint = v
		}
		if v := endpoint.ResourceManagerAudience.ValueString(); v != "" {
			resourceManagerAudience = v
		}
		cloudConfig.Services[cloud.ResourceManager] = cloud.ServiceConfiguration{
			Endpoint: resourceManagerEndpoint,
			Audience: resourceManagerAudience,
		}
		if v := endpoint.ActiveDirectoryAuthorityHost.ValueString(); v != "" {
			cloudConfig.ActiveDirectoryAuthorityHost = v
		}
	}

	var auxTenants []string
	if elements := model.AuxiliaryTenantIDs.Elements(); len(elements) != 0 {
		for _, element := range elements {
			auxTenants = append(auxTenants, element.(basetypes.StringValue).ValueString())
		}
	}

	option := azidentity.DefaultAzureCredentialOptions{
		AdditionallyAllowedTenants: auxTenants,
		ClientOptions: azcore.ClientOptions{
			Cloud: cloudConfig,
		},
		TenantID: model.TenantID.ValueString(),
	}

	cred, err := buildChainedTokenCredential(model, option)
	if err != nil {
		response.Diagnostics.AddError("Failed to obtain a credential.", err.Error())
		return
	}

	copt := &clients.Option{
		Cred:                 cred,
		CloudCfg:             cloudConfig,
		ApplicationUserAgent: buildUserAgent(request.TerraformVersion, model.PartnerID.ValueString(), model.DisableTerraformPartnerID.ValueBool()),
		Features: features.UserFeatures{
			DefaultTags:          tags.ExpandTags(model.DefaultTags),
			DefaultLocation:      location.Normalize(model.DefaultLocation.ValueString()),
			DefaultNaming:        model.DefaultName.ValueString(),
			EnablePreflight:      model.EnablePreflight.ValueBool(),
			DisableDefaultOutput: model.DisableDefaultOutput.ValueBool(),
		},
		SkipProviderRegistration:    model.SkipProviderRegistration.ValueBool(),
		DisableCorrelationRequestID: model.DisableCorrelationRequestID.ValueBool(),
		CustomCorrelationRequestID:  model.CustomCorrelationRequestID.ValueString(),
		SubscriptionId:              model.SubscriptionID.ValueString(),
		TenantId:                    model.TenantID.ValueString(),
	}

	client := &clients.Client{}
	if err = client.Build(ctx, copt); err != nil {
		response.Diagnostics.AddError("Error Building Client", err.Error())
		return
	}

	// load schema
	azure.GetAzureSchema()

	response.ResourceData = client
	response.DataSourceData = client
	response.EphemeralResourceData = client
}

func (p Provider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		func() function.Function {
			return &functions.ParseResourceIdFunction{}
		},
		func() function.Function { return &functions.BuildResourceIdFunction{} },
		func() function.Function {
			return &functions.TenantResourceIdFunction{}
		},
		func() function.Function {
			return &functions.SubscriptionResourceIdFunction{}
		},
		func() function.Function { return &functions.ResourceGroupResourceIdFunction{} },
		func() function.Function { return &functions.ManagementGroupResourceIdFunction{} },
		func() function.Function { return &functions.ExtensionResourceIdFunction{} },
		func() function.Function { return &functions.UniqueStringFunction{} },
	}
}

func (p Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return &services.ResourceIdDataSource{}
		},
		func() datasource.DataSource {
			return &services.ResourceListDataSource{}
		},
		func() datasource.DataSource {
			return &services.ResourceActionDataSource{}
		},
		func() datasource.DataSource {
			return &services.AzapiResourceDataSource{}
		},
		func() datasource.DataSource {
			return &services.ClientConfigDataSource{}
		},
	}

}

func (p Provider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return &services.AzapiResource{}
		},
		func() resource.Resource {
			return &services.AzapiUpdateResource{}
		},
		func() resource.Resource {
			return &services.ActionResource{}
		},
		func() resource.Resource {
			return &services.DataPlaneResource{}
		},
	}
}

func (p Provider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		func() ephemeral.EphemeralResource {
			return &services.ActionEphemeral{}
		},
	}
}

func buildUserAgent(terraformVersion string, partnerID string, disableTerraformPartnerID bool) string {
	if terraformVersion == "" {
		// Terraform 0.12 introduced this field to the protocol
		// We can therefore assume that if it's missing it's 0.10 or 0.11
		terraformVersion = "0.11+compatible"
	}
	tfUserAgent := fmt.Sprintf("HashiCorp Terraform/%s (+https://www.terraform.io)", terraformVersion)
	providerUserAgent := fmt.Sprintf("terraform-provider-azapi/%s", version.ProviderVersion)
	userAgent := strings.TrimSpace(fmt.Sprintf("%s %s", tfUserAgent, providerUserAgent))

	// append the CloudShell version to the user agent if it exists
	if azureAgent := os.Getenv("AZURE_HTTP_USER_AGENT"); azureAgent != "" {
		userAgent = fmt.Sprintf("%s %s", userAgent, azureAgent)
	}

	// only one pid can be interpreted currently
	// hence, send partner ID if present, otherwise send Terraform GUID
	// unless users have opted out
	if partnerID == "" && !disableTerraformPartnerID {
		// Microsoft’s Terraform Partner ID is this specific GUID
		partnerID = "222c6c49-1b0a-5959-a213-6608f9eb8820"
	}

	if partnerID != "" {
		userAgent = fmt.Sprintf("%s pid-%s", userAgent, partnerID)
	}
	return userAgent
}

func buildChainedTokenCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, error) {
	log.Printf("[DEBUG] building chained token credential")
	var creds []azcore.TokenCredential

	if model.UseOIDC.ValueBool() || model.UseAKSWorkloadIdentity.ValueBool() {
		log.Printf("[DEBUG] oidc credential or AKS Workload Identity enabled")
		if cred, err := buildOidcCredential(model, options); err == nil {
			creds = append(creds, cred)
		} else {
			log.Printf("[DEBUG] failed to initialize oidc credential: %v", err)
		}

		log.Printf("[DEBUG] azure pipelines credential enabled")
		if cred, err := buildAzurePipelinesCredential(model, options); err == nil {
			creds = append(creds, cred)
		} else {
			log.Printf("[DEBUG] failed to initialize azure pipelines credential: %v", err)
		}
	}

	if cred, err := buildClientSecretCredential(model, options); err == nil {
		creds = append(creds, cred)
	} else {
		log.Printf("[DEBUG] failed to initialize client secret credential: %v", err)
	}

	if cred, err := buildClientCertificateCredential(model, options); err == nil {
		creds = append(creds, cred)
	} else {
		log.Printf("[DEBUG] failed to initialize client certificate credential: %v", err)
	}

	if model.UseMSI.ValueBool() {
		log.Printf("[DEBUG] msi credential enabled")
		if cred, err := buildManagedIdentityCredential(model, options); err == nil {
			creds = append(creds, cred)
		} else {
			log.Printf("[DEBUG] failed to initialize msi credential: %v", err)
		}
	}

	if model.UseCLI.ValueBool() {
		log.Printf("[DEBUG] cli credential enabled")
		if cred, err := buildAzureCLICredential(options); err == nil {
			creds = append(creds, cred)
		} else {
			log.Printf("[DEBUG] failed to initialize cli credential: %v", err)
		}
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials were successfully initialized")
	}

	return azidentity.NewChainedTokenCredential(creds, nil)
}

func buildClientSecretCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building client secret credential")
	clientID, err := model.GetClientId()
	if err != nil {
		return nil, err
	}
	clientSecret, err := model.GetClientSecret()
	if err != nil {
		return nil, err
	}
	o := &azidentity.ClientSecretCredentialOptions{
		AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
		ClientOptions:              options.ClientOptions,
		DisableInstanceDiscovery:   options.DisableInstanceDiscovery,
	}
	return azidentity.NewClientSecretCredential(options.TenantID, *clientID, *clientSecret, o)
}

func buildClientCertificateCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building client certificate credential")
	clientID, err := model.GetClientId()
	if err != nil {
		return nil, err
	}

	var certData []byte
	if certPath := model.ClientCertificatePath.ValueString(); certPath != "" {
		log.Printf("[DEBUG] reading certificate from file %s", certPath)
		// #nosec G304
		certData, err = os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf(`failed to read certificate file "%s": %v`, certPath, err)
		}
	}
	if certBase64 := model.ClientCertificate.ValueString(); certBase64 != "" {
		log.Printf("[DEBUG] decoding certificate from base64")
		certData, err = decodeCertificate(certBase64)
		if err != nil {
			return nil, err
		}
	}

	if len(certData) == 0 {
		return nil, fmt.Errorf("no certificate data provided")
	}

	var password []byte
	if v := model.ClientCertificatePassword.ValueString(); v != "" {
		password = []byte(v)
	}
	certs, key, err := azidentity.ParseCertificates(certData, password)
	if err != nil {
		return nil, fmt.Errorf(`failed to load certificate": %v`, err)
	}
	o := &azidentity.ClientCertificateCredentialOptions{
		AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
		ClientOptions:              options.ClientOptions,
		DisableInstanceDiscovery:   options.DisableInstanceDiscovery,
	}
	return azidentity.NewClientCertificateCredential(options.TenantID, *clientID, certs, key, o)
}

func buildOidcCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building oidc credential")
	clientId, err := model.GetClientId()
	if err != nil {
		return nil, err
	}
	if model.OIDCToken.ValueString() == "" && model.GetOIDCTokenFilePath() == "" && (model.OIDCRequestToken.ValueString() == "" || model.OIDCRequestURL.ValueString() == "") {
		return nil, fmt.Errorf("missing required OIDC configuration")
	}
	o := &OidcCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: options.Cloud,
		},
		AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
		TenantID:                   options.TenantID,
		ClientID:                   *clientId,
		RequestToken:               model.OIDCRequestToken.ValueString(),
		RequestUrl:                 model.OIDCRequestURL.ValueString(),
		Token:                      model.OIDCToken.ValueString(),
		TokenFilePath:              model.GetOIDCTokenFilePath(),
	}
	return NewOidcCredential(o)
}

func buildManagedIdentityCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building managed identity credential")
	clientId, err := model.GetClientId()
	if err != nil {
		return nil, err
	}
	o := &azidentity.ManagedIdentityCredentialOptions{
		ClientOptions: options.ClientOptions,
		ID:            azidentity.ClientID(*clientId),
	}
	return NewManagedIdentityCredential(o)
}

func buildAzureCLICredential(options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building azure cli credential")
	o := &azidentity.AzureCLICredentialOptions{
		AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
		TenantID:                   options.TenantID,
	}
	return azidentity.NewAzureCLICredential(o)
}

func buildAzurePipelinesCredential(model providerData, options azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	log.Printf("[DEBUG] building azure pipeline credential")
	o := &azidentity.AzurePipelinesCredentialOptions{
		ClientOptions:              options.ClientOptions,
		AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
		DisableInstanceDiscovery:   options.DisableInstanceDiscovery,
	}
	clientId, err := model.GetClientId()
	if err != nil {
		return nil, err
	}
	return azidentity.NewAzurePipelinesCredential(options.TenantID, *clientId, model.OIDCAzureServiceConnectionID.ValueString(), model.OIDCRequestToken.ValueString(), o)
}

func decodeCertificate(clientCertificate string) ([]byte, error) {
	var pfx []byte
	if clientCertificate != "" {
		out := make([]byte, base64.StdEncoding.DecodedLen(len(clientCertificate)))
		n, err := base64.StdEncoding.Decode(out, []byte(clientCertificate))
		if err != nil {
			return pfx, fmt.Errorf("could not decode client certificate data: %v", err)
		}
		pfx = out[:n]
	}
	return pfx, nil
}
