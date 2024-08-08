package functions_test

import (
	"context"
	"testing"

	"github.com/Azure/terraform-provider-azapi/internal/services/functions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Test_SubscriptionResourceIdFunction(t *testing.T) {
	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"subscription-scope-valid-single": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue("Microsoft.Sql/locations@2015-05-01-preview"),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("loc1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.ObjectValueMust(functions.BuildResourceIdResultAttrTypes, map[string]attr.Value{
					"resource_id": types.StringValue("/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Sql/locations/loc1"),
				})),
			},
		},
		"subscription-scope-valid-double": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue("Microsoft.Sql/locations/usages@2015-05-01-preview"),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("loc1"),
						types.StringValue("usage1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.ObjectValueMust(functions.BuildResourceIdResultAttrTypes, map[string]attr.Value{
					"resource_id": types.StringValue("/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Sql/locations/loc1/usages/usage1"),
				})),
			},
		},
		"subscription-scope-invalid-single": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue("Microsoft.Sql/locations@2015-05-01-preview"),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("loc1"),
						types.StringValue("usage1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Error:  function.NewFuncError("number of resource names does not match the number of resource type parts, expected 1, got 2"),
				Result: function.NewResultData(types.ObjectUnknown(functions.BuildResourceIdResultAttrTypes)),
			},
		},
		"subscription-scope-invalid-double": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue("Microsoft.Sql/locations/usages@2015-05-01-preview"),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("loc1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Error:  function.NewFuncError("number of resource names does not match the number of resource type parts, expected 2, got 1"),
				Result: function.NewResultData(types.ObjectUnknown(functions.BuildResourceIdResultAttrTypes)),
			},
		},
		"subscription-scope-invalid-empty-type": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue(""),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("rg1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Error:  function.NewFuncError("`type` is invalid, it should be like `ResourceProvider/resourceTypes@ApiVersion`"),
				Result: function.NewResultData(types.ObjectUnknown(functions.BuildResourceIdResultAttrTypes)),
			},
		},
		"subscription-scope-invalid-type": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("00000000-0000-0000-0000-000000000000"),
					types.StringValue("Invalid/ResourceType"),
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("rg1"),
					}),
				}),
			},
			expected: function.RunResponse{
				Error:  function.NewFuncError("`type` is invalid, it should be like `ResourceProvider/resourceTypes@ApiVersion`"),
				Result: function.NewResultData(types.ObjectUnknown(functions.BuildResourceIdResultAttrTypes)),
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(functions.BuildResourceIdResultAttrTypes)),
			}

			subscriptionResourceIdFunction := functions.SubscriptionResourceIdFunction{}
			subscriptionResourceIdFunction.Run(context.Background(), testCase.request, &got)

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}
