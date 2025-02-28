package services

import (
	"encoding/json"
	"errors"

	"github.com/Azure/terraform-provider-azapi/internal/services/dynamic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildOutputFromBody(responseBody interface{}, modelResponseExportValues types.Dynamic, defaultResult interface{}) (types.Dynamic, error) {
	if modelResponseExportValues.IsNull() {
		if defaultResult == nil {
			return types.DynamicValue(types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})), nil
		}
		data, err := json.Marshal(defaultResult)
		if err != nil {
			return types.DynamicNull(), err
		}
		return dynamic.FromJSONImplied(data)
	}

	data, err := dynamic.ToJSON(modelResponseExportValues)
	if err != nil {
		return types.DynamicNull(), err
	}

	switch modelResponseExportValues.UnderlyingValue().(type) {
	case types.List, types.Tuple, types.Set:
		var responseExportValues []string
		if err = json.Unmarshal(data, &responseExportValues); err != nil {
			return types.DynamicNull(), err
		}

		return types.DynamicValue(flattenOutput(responseBody, responseExportValues)), nil
	case types.Map, types.Object:
		var responseExportValues map[string]string
		if err = json.Unmarshal(data, &responseExportValues); err != nil {
			return types.DynamicNull(), err
		}

		return types.DynamicValue(flattenOutputJMES(responseBody, responseExportValues)), nil
	default:
		return types.DynamicNull(), errors.New("unsupported type for response_export_values, must be a list or map")
	}
}

func volatileFieldList() []string {
	return []string{
		"etag",
		"updatedBy",
		"updated",
		"updatedOn",
		"updatedTimestamp",
		"lastUpdatedOn",
		"lastUpdated",
		"lastUpdatedTime",
		"lastUpdatedTimeUtc",
		"lastUpdatedDateUTC",
		"modifiedOn",
		"lastModifiedUtc",
		"lastModifiedTimeUtc",
		"lastModifiedAt",
		"lastModifiedBy",
		"lastModifiedByType",
		"freeTrialRemainingTime",
		"trialDaysRemaining",
		"daysTrialRemaining",
	}
}
