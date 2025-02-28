## v2.3.0 (unreleased)
FEATURES:
- **New Ephemeral Resource**: azapi_resource_action

ENHANCEMENTS:
- `azapi` provider: The `oidc_azure_service_connection_id` field can be sourced from the `ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID` or `ARM_OIDC_AZURE_SERVICE_CONNECTION_ID` Environment Variables.
- `azapi` provider: The `enable_preflight` field can be sourced from the `ARM_ENABLE_PRE_FLIGHT` Environment Variable.
- `azapi_resource_action` resource, data source: Support `sensitive_response_export_values` field, which is used to specify the sensitive fields to export.
- `azaapi_resource_action` resource, data source: Support `sensitive_output` field, which is a sensitive computed field that contains the fields exported by `sensitive_response_export_values`.

BUG FIXES:
- Fix a bug that query parameters and headers don't work properly with unknown values
- Fix more edge cases that the provider produced inconsistent result after apply when default output feature is enabled.
- Fix a bug that when moving resource from `azurerm` resource, the id could not be parsed correctly.


## v2.2.0

ENHANCEMENTS:
- `azapi_resource` resource: Improved 403 handling for management group resources to include child resources
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/4da2e194de989ed72552add82b9a5ead5223695b

BUG FIXES:
- Fix a bug that the provider produced inconsistent result after apply when default output feature is enabled.  
Notice: Terraform will detect the `output` field's changes made outside of Terraform since the last "terraform apply". You can run `terraform refresh` to update the state file with the latest values.
- Fix a bug that the GET after PUT retry timeout doesn't work properly when the environment variable is set.

## v2.1.0
FEATURES:
- `azapi_resource` resource: Support resource move operation, it allows moving resources from `azurerm` provider.

ENHANCEMENTS:
- `azapi_client_config` data source: Support `object_id` field.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/401bed53e5495fb79f6c3357d9befb9fea158b1f

BUG FIXES:
- Fix a bug when `body` contains an unknown float number, the provider will crash.
- Fix the crash that occurs when no tenant ID is configured in Azure CLI.
- Fix a bug that using multiple locks can result in a deadlock.

## v2.0.1
BREAKING CHANGES:
- `azapi_resource`, `azapi_update_resource` resources and data sources' `output` field defaults to the readonly fields when the `response_export_values` is not specified.
- `azapi_resource_list` data source's `output` field defaults to the response when the `response_export_values` is not specified.

ENHANCEMENTS:
- `azapi_data_plane_resource` resource: Support `Microsoft.Purview/accounts/Scanning/managedvirtualnetworks` type.
- Support a default retry policy that retries when GET request fails with 404 status code after resource creation.
- `azapi` provider: Support `disable_default_output` field, which is used to disable the default output for the resources and data sources.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/c3ff45dfffe7f229447639b5982a1e2deadc1b71

BUG FIXES:
- Fix a bug that non-standard LRO error responses are not handled properly.

## v2.0.0-beta
BREAKING CHANGES:
- Provider field `default_naming_prefix` and `default_naming_suffix` are removed. Please specify the naming prefix and suffix in the resource's `name` field instead.
- Provider field `enable_hcl_output_for_data_source` is removed. The `output` field in the data source is always in HCL format.
- The `azapi_resource`'s `removing_special_chars` field is removed. Please specify the `name` field and remove the special characters in the `name` field instead.
- The `ignore_body_changes` field is removed. Please use the `lifecyle.ignore_changes` to ignore some properties when comparing the resource with its current state.
- The `body` field now only accepts an HCL object. Please remove the `jsonencode` function when using the `body` field.
- The `output` field now only exports an HCL object. Please remove the `jsondecode` function when using the `output` field.
- The `use_msi` field now defaults to `false`, please set it to `true` explicitly if you want to authenticate using Managed Service Identity.

FEATURES:
- **New Provider Function**: build_resource_id
- **New Provider Function**: parse_resource_id
- **New Provider Function**: subscription_resource_id
- **New Provider Function**: tenant_resource_id
- **New Provider Function**: management_group_resource_id
- **New Provider Function**: resource_group_resource_id
- **New Provider Function**: extension_resource_id

ENHANCEMENTS:
- `azapi` provider: Support `enable_preflight` field, which is used to enable Preflight Validation, the default value is `false`.
- `azapi_resource` and `azapi_data_plane_resource` resource: Support `replace_triggers_external_values` field which is used to trigger a replacement of the resource.
- `azapi_resource` and `azapi_data_plane_resource` resource: Support `replace_triggers_refs` field which is used to trigger a replacement of the resource.
- `azapi` resources and data sources: Support `retry` field, which is used to specify the retry configuration.
- `azapi` resources and data sources: Support `headers` and `query_parameters` fields, which are used to specify the headers and query parameters.
- `azapi` resources and data sources: The `response_export_values` field supports JMESPath expressions.
- Accept `AZURE_CLIENT_ID` and `AZURE_TENANT_ID` environment variables when authenticating using AKS workload identity.
- `azapi` provider: Support `oidc_azure_service_connection_id` field, which is used to specify the Azure Service Connection ID for OIDC authentication with Azure DevOps.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/7492c6d0a12a07f97b955661bf6df83d51bbb14d


## v1.15.0

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/5ccee7fe1b353e40ed86bfc530ee185faa43a288

BUG FIXES:
- Fix a bug that differences in a list of objects are not detected correctly.
- Fix a bug that azapi_update_resource produced inconsistent results when the case of the `resource_id` and `type` fields are different.
- Fix a bug that provider will crash when there's no default subscription configured in azure cli.
- Fix a bug that resource definitions are not correctly cached.

## v1.14.0
FEATURES:
- **New Data Source**: azapi_client_config

ENHANCEMENTS:
- `azapi` provider: Support `client_certificate` field, which specifies base64-encoded PKCS#12 bundle to be used as the client certificate for authentication.
- `azapi` provider: Support `use_aks_workload_identity` field to enable AKS workload identity.
- `azapi_resource`, `azapi_update_resource`, `azapi_resource_action`, `azapi_data_plane_resource` resources: Support `timeouts.update` field, which is used to specify the timeout for the update operation.
- `azapi_update_resource` resource: Improve the id build logic to honor user's input.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/37dcb1890e3661255614961f470676b486272ff2

BUG FIXES:
- Fix a bug that `azapi_resource` will crash when the `location` in GET response is null.
- Fix a bug that schema validation fails to validate unknown string values.
- Fix a bug that `azapi_resource` only supports extension resource on a resource group scoped resource.
- Fix a bug that OIDC authentication should not be used when required fields are missing.

## v1.13.1

ENHANCEMENTS:
- `azapi` provider: Support `enable_hcl_output_for_data_source` field, which is used to enable the HCL output for the data source, the default value is `false`.
  This could resolve the following breaking changes in the previous release:
- `azapi_resource` data source: The `output` field changes from JSON string to HCL object. Users can use access the fields in the output as an HCL object. Please remove the `jsondecode` function when using the `output` field.
- `azapi_resource_list` data source: The `output` field changes from JSON string to HCL object. Users can use access the fields in the output as an HCL object. Please remove the `jsondecode` function when using the `output` field.

BUG FIXES:
- Fix a bug when upgrading from previous provider `azapi_resource` resource will set `tags` for resources that don't have `tags` in the configuration.
- Fix a bug that `azapi_resource` resource cannot handle tags with unknown values.
- Fix a bug that `null` string value can't pass the schema validation.
- Fix a bug that required fields which have `null` value can't pass the schema validation.
- Fix a bug that schema validation fails to validate the float number in the body.
- Fix a bug that client certificate authentication doesn't work.
- Fix a bug that auxiliary tenant ids are not passed to the client.
- Fix a bug that `azapi_resource_action` resource shows the plan diff when upgrading from the previous provider.

## v1.13.0
BREAKING CHANGES:
- Provider field `default_naming_prefix` and `default_naming_suffix` are deprecated. It will not work in this release and will be removed in the next major release.
  Please specify the naming prefix and suffix in the resource's `name` field instead.
- The `azapi_resource`'s `removing_special_chars` field is deprecated. It will not work in this release and will be removed in the next major release.
  Please specify the `name` field and remove the special characters in the `name` field instead.
- Defining the `identity` inside the `body` field is not recommended. In this release, it will not sync the `identity` inside the `body` field to `identity` block.
  Please define the `identity` block instead.
- `azapi_resource` data source: The `output` field changes from JSON string to HCL object. Users can use access the fields in the output as an HCL object. Please remove the `jsondecode` function when using the `output` field.
- `azapi_resource_list` data source: The `output` field changes from JSON string to HCL object. Users can use access the fields in the output as an HCL object. Please remove the `jsondecode` function when using the `output` field.

ENHANCEMENTS:
- `azapi_resource` resource, `azapi_update_resource` resource, `azapi_resource_action` resource, `azapi_data_plane_resource` resource, `azapi_resource_action` data source: The `body` field supports the dynamic schema and allows user to use the HCL object to specify the body.
- `azapi_resource` resource, `azapi_update_resource` resource, `azapi_resource_action` resource, `azapi_data_plane_resource` resource, `azapi_resource_action` data source, `azapi_resource` data source, `azapi_resource_list` data source: The `output` field supports the dynamic schema and allows user to read the output as an HCL object.
- `azapi` provider: Support `client_id_file_path`and `client_secret_file_path` fields, which are used to specify the file path of the client id and client secret.
- `azapi_data_plane_resource` resource: Support `Microsoft.Synapse/workspaces/databases` type.
- `azapi_resource` resource, `azapi_update_resource` resource: Ignore the order of the elements in a list if the element has a `name` field as identifier.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/48ce933ad85391b60ee02cf83e17c9b28d31a7b1

BUG FIXES:
- Fix a bug that `azapi_resource_action` doesn't support 204 status code as a success response.


## v1.12.1
BUG FIXES:
- Fix a bug that `azapi_resource` removes some required fields in the `body` when using `ignore_body_changes` feature.

## v1.12.0
ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/4abd79ba2baa05ba3c8364919b670ab43a9bf69c

BUG FIXES:
- Fix a bug that `ignore_body_changes` incorrectly removes tags. 

NOTICE:
- Provider field `default_naming_prefix` and `default_naming_suffix` are deprecated. It will not work in the next minor release and will be removed in the next major release.
  Please specify the naming prefix and suffix in the resource's `name` field instead.
- The `azapi_resource`'s `removing_special_chars` field is deprecated. It will not work in the next minor release and will be removed in the next major release.
  Please specify the `name` field and remove the special characters in the `name` field instead.
- Defining the `identity` inside the `body` field is not recommended. In the next minor release, it will not sync the `identity` inside the `body` field to `identity` block.
  Please define the `identity` block instead.

## v1.11.0
ENHANCEMENTS:
- `azapi_resource_action` resource: Support `HEAD` method.
- `azapi_resource_action` resource: Support `when` field, which is used to control when to execute the action.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/fcfe2a66a04575f204767182fc575612c82eebc1


## v1.10.0
ENHANCEMENTS:
- `azapi_resource` data source: When creating `Microsoft.Resources/subscriptions`, `resource_id` is optional and defaults to the ID of the default subscription.
- Add a new logger to record the traffic in a structured way.
- `azapi`: Support `endpoint` block, which is used to configure the endpoints of the Azure Clouds.
- `azapi_resource_action` resource: Support `GET` method.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/505b813ce50368156e3da1b86f07977b5a913be9

BUG FIXES:
- Fix a bug that `body` is not set when import with an unrecognized `api-version`.
- Fix a bug that deploy time constants are not removed from the request body when using `azapi_update_resource` resource.

## v1.9.0
FEATURES:
- **New Data Source**: azapi_resource_list
- **New Data Source**: azapi_resource_id

ENHANCEMENTS:
- `azapi_resource` resource/data source: When creating `Microsoft.Resources/resourceGroups`, `parent_id` is optional and defaults to the ID of the default subscription.
- `azapi_resource` resource: Support `ignore_body_changes` field, which is used to ignore some properties when comparing the resource with its current state.
- `azapi_update_resource` resource: Support `ignore_body_changes` field, which is used to ignore some properties when comparing the resource with its current state.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/1d8fec8184258cdf967b1288b156e01f7cbc8ca9

BUG FIXES:
- Fix a bug that `azapi_resource` resource doesn't store the `id` in the state when error happens during the creation.
- Fix a bug that errors from the polling API which don't follow the ARM LRO guidelines are not handled properly.

## v1.8.0
FEATURES:

ENHANCEMENTS:
- `azapi_resource_action`: Support provider action.
- Update bicep types to https://ms-henglu/bicep-types-az/commit/c616eb1ad4980f63c0d6b436a63701e175a62224

BUG FIXES:
- Fix a bug that resource id for type `Microsoft.Resources/providers` is not parsed correctly.
- Fix a bug that resource id for type `Microsoft.Resources/tenants` is not parsed correctly.

## v1.7.0
FEATURES:
- **New Resource**: azapi_data_plane_resource
- `azapi`: Support `use_msi` and `use_cli` features.
- `azapi`: Support `auxiliary_tenant_ids` field, which is required for multi-tenancy and cross-tenant scenarios.
- `azapi`: Support `custom_correlation_request_id` field, which is used to specify the correlation request id.

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/0536b68e779fba100b9fbe32737c38d75396e2cf

BUG FIXES:
- Fix a bug that provider crashes when loading azure schema.

## v1.6.0
FEATURES:

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/da15d0376faa02a6e891dee315910535cef2a13f

BUG FIXES:
- Fix the bug that the headers are not stored in the log.

## v1.5.0
FEATURES:

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/b8626aecc5f47b70086580956adfcd1e267a49e6

BUG FIXES:

## v1.4.0
FEATURES:
- `azapi`: Support `default_name`, `default_naming_prefix` and `defualt_naming_suffix` features.

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/a915acab5788d890aed774ec818535b44311d16d

BUG FIXES:
- Fix a bug that when apply failed there still are some attributes stored in the state.
- Fix a bug that schema validation requires redundant `name` fields both in resource and in body.

## v1.3.0
FEATURES:
- `azapi`: Support OIDC authentication.

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/78ec1b99699a4bf44869bd13f1b0ed7d92a99c27
- `azapi_resource`: `ignore_missing_property` will also ignore the sensitive properties returned in asterisks.

BUG FIXES:
- Fix a document typo.

## v1.2.0
FEATURES:
- `azapi`: Support `client_certificate_password` option.

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/019b2d62fe84508582b8c54ce3d91c2b4840e624

BUG FIXES:

## v1.1.0
FEATURES:

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/e641570bedc5004498d3e374adb60fdfd3521b09

BUG FIXES:
- `azapi_resource_action`: The `output` is not refreshed when `body` is changed.

## v1.0.0
FEATURES:

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/a6dabb0cd645c17a1accf3ec1be4d7930e982b23

BUG FIXES:

BREAKING CHANGES:
- `azapi_resource`: `ignore_missing_property`'s default value changed from `false` to `true`.
- `azapi_update_resource`: `ignore_missing_property`'s default value changed from `false` to `true`.

## v0.6.0
FEATURES:

ENHANCEMENTS:
- `azapi_resource_action`: Supports `locks` which used to prevent modifying resources at the same time.
- `azapi_resource_action`: Supports parse response which `Content-Type` is `text/plain`.
- Improve validation on `type`, `parent_id` and `resource_id`.
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/5451fcd5e1bf4d8313d475d8e3dc28efc7a77e2a

BUG FIXES:

## v0.5.0
FEATURES:
- **New Data Source**: azapi_resource_action
- **New Resource**: azapi_resource_action

ENHANCEMENTS:
- Update bicep types to https://github.com/ms-henglu/bicep-types-az/commit/813d8bbc9ecf432a2a0ff2769627592fae34369f

BUG FIXES:
- DefaultAzureCredential authentication failed because empty clientId is set

## v0.4.0
FEATURES:

ENHANCEMENTS:
- `azapi_resource`: Supports default api-version when importing existing resource into terraform state.
- `azapi_resource`: Supports `locks` which used to prevent creating/modifying/deleting resources at the same time.
- `azapi_update_resource`: Supports `locks` which used to prevent creating/modifying/deleting resources at the same time.
- `azapi_resource` data source: Supports configuring `resource_id` as an alternative way to configure `name` and `parent_id`.
- `azapi` provider: Supports `partner_id`, `disable_terraform_partner_id` and `disable_terraform_partner_id`.
- Update bicep types to https://github.com/Azure/bicep-types-az/commit/ea703e2aba0d1c024f33124ee2cd34bc0c6084b5

BUG FIXES:

## v0.3.0
FEATURES:

ENHANCEMENTS:
- Update bicep types to https://github.com/Azure/bicep-types-az/commit/644ff521c92ce8d493f6da977af12377f32abffc

BUG FIXES:

## v0.2.1
FEATURES:

ENHANCEMENTS:

BUG FIXES:
- Improve error message for schema validation failure.
- DefaultAzureCredential reads the client ID of a user-assigned managed identity.
- Fix the modification is not working, when use `azapi_update_resource` to modify additional properties.
- Fix crash when use `azapi_update_resource` on a resource whose id is null
- Fix crash when the discriminated type is not in the embedded schema

## v0.2.0
FEATURES:

ENHANCEMENTS:
- Setting `response_export_values = ["*"]` will export the full response body.
- Update bicep types to https://github.com/Azure/bicep-types-az/commit/57f3ecc750648562cf170ef456ef39533872b101

BUG FIXES:
- Fix incorrect ID format in the imported `azapi_resource` resource. 
- Fix incorrect `body` content in the imported `azapi_resource` resource.

## v0.1.1

FEATURES:

ENHANCEMENTS:

BUG FIXES:

- Fix document format.

## v1.1.0

FEATURES:
- **New Data Source**: azapi_resource
- **New Resource**: azapi_resource
- **New Resource**: azapi_update_resource
- **Provider feature**: support default location and default tags

ENHANCEMENTS:

BUG FIXES:
