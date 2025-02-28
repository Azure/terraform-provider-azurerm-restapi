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

func Test_UniqueStringFunction(t *testing.T) {
	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"unique-string-valid": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("/subscriptions/00000000-0000-0000-0000-000000000000"),
						types.StringValue("resource-id"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("cwvxuqg24sifi")),
			},
		},
		"unique-string-fu": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("fu"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("6rkxbspxjmsho")),
			},
		},
		"unique-string-fubar": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("fubar"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("cj2xpqsiwjfne")),
			},
		},
		"unique-string-fu-bar": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("fu"),
						types.StringValue("bar"),
					}),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("q5wxoscxs5j6k")),
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got := function.RunResponse{
				Result: function.NewResultData(types.StringUnknown()),
			}

			uniqueStringFunction := functions.UniqueStringFunction{}
			uniqueStringFunction.Run(context.Background(), testCase.request, &got)

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}
