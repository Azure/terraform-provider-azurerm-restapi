package parse_test

import (
	"testing"

	"github.com/Azure/terraform-provider-azapi/internal/services/parse"
)

func Test_AzurermIdToAzureId(t *testing.T) {
	testcases := []struct {
		name                string
		azurermResourceType string
		azurermId           string
		expectedAzureId     string
		expectError         bool
	}{
		{
			name:                "azurerm_monitor_diagnostic_setting valid",
			azurermResourceType: "azurerm_monitor_diagnostic_setting",
			azurermId:           "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm|diagnosticSettingName",
			expectedAzureId:     "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm/providers/Microsoft.Insights/diagnosticSettings/diagnosticSettingName",
			expectError:         false,
		},
		{
			name:                "azurerm_monitor_diagnostic_setting invalid",
			azurermResourceType: "azurerm_monitor_diagnostic_setting",
			azurermId:           "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm|diagnosticSettingName|foo",
			expectedAzureId:     "",
			expectError:         true,
		},
		{
			name:                "azurerm_role_definition valid",
			azurermResourceType: "azurerm_role_definition",
			azurermId:           "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/roleDefinitions/roleDefinitionId|/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg",
			expectedAzureId:     "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/roleDefinitions/roleDefinitionId",
			expectError:         false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			azureId, err := parse.AzurermIdToAzureId(tc.azurermResourceType, tc.azurermId)
			if (err != nil) != tc.expectError {
				t.Fatalf("expected error: %v, got: %v", tc.expectError, err)
			}
			if azureId != tc.expectedAzureId {
				t.Fatalf("expected azure id: %s, got: %s", tc.expectedAzureId, azureId)
			}
		})
	}

}
