package parse

import (
	"fmt"
	"strings"
)

// AzurermIdToAzureId converts an azurerm id to an azure resource id
func AzurermIdToAzureId(azurermResourceType string, azurermId string) (string, error) {
	switch azurermResourceType {
	case "azurerm_monitor_diagnostic_setting":
		// input: <target id>|<diagnostic setting name>
		// output: <target id>/providers/Microsoft.Insights/diagnosticSettings/<diagnostic setting name>
		azurermIdSplit := strings.Split(azurermId, "|")
		if len(azurermIdSplit) != 2 {
			return "", fmt.Errorf("invalid id: %s, expected format: <target id>|<diagnostic setting name>", azurermId)
		}
		return fmt.Sprintf("%s/providers/Microsoft.Insights/diagnosticSettings/%s", azurermIdSplit[0], azurermIdSplit[1]), nil
	case "azurerm_role_definition":
		// input: <role definition id>|<scope>
		// output: <role definition id>
		azurermIdSplit := strings.Split(azurermId, "|")
		if len(azurermIdSplit) != 2 {
			return "", fmt.Errorf("invalid id: %s, expected format: <role definition id>|<scope>", azurermId)
		}
		return azurermIdSplit[0], nil

		// add more cases here as needed
	}
	// return azure id
	return azurermId, nil
}
