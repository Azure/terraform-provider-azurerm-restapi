package clients

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/terraform-provider-azapi/internal/retry"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestConfigureCustomRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name                    string
		rtry                    retry.RetryValue
		useReadAfterCreate      bool
		expectedInitialInterval time.Duration
		expectedMaxInterval     time.Duration
		expectedMultiplier      float64
		expectedRandomization   float64
		expectedStatusCodes     []int
		expectedErrorRegexps    []string
	}{
		{
			name:                    "default retry configuration",
			rtry:                    retry.NewRetryValueNull(),
			useReadAfterCreate:      false,
			expectedInitialInterval: retry.DefaultIntervalSeconds * time.Second,
			expectedMaxInterval:     retry.DefaultMaxIntervalSeconds * time.Second,
			expectedMultiplier:      retry.DefaultMultiplier,
			expectedRandomization:   retry.DefaultRandomizationFactor,
			expectedStatusCodes:     retry.NewRetryValueNull().GetDefaultRetryableStatusCodes(),
			expectedErrorRegexps:    []string{},
		},
		{
			name:                    "default retry configuration with read after create",
			rtry:                    retry.NewRetryValueNull(),
			useReadAfterCreate:      true,
			expectedInitialInterval: retry.DefaultIntervalSeconds * time.Second,
			expectedMaxInterval:     retry.DefaultMaxIntervalSeconds * time.Second,
			expectedMultiplier:      retry.DefaultMultiplier,
			expectedRandomization:   retry.DefaultRandomizationFactor,
			expectedStatusCodes:     retry.NewRetryValueNull().GetDefaultRetryableReadAfterCreateStatusCodes(),
			expectedErrorRegexps:    []string{},
		},
		{
			name: "custom retry configuration",
			rtry: retry.NewRetryValueMust(retry.NewRetryValueNull().AttributeTypes(ctx), map[string]attr.Value{
				"interval_seconds":     basetypes.NewInt64Value(10),
				"max_interval_seconds": basetypes.NewInt64Value(60),
				"multiplier":           basetypes.NewFloat64Value(1.2),
				"randomization_factor": basetypes.NewFloat64Value(0.2),
				"error_message_regex":  basetypes.NewListValueMust(types.StringType, []attr.Value{basetypes.NewStringValue("timeout"), basetypes.NewStringValue("temporary")}),
			}),
			useReadAfterCreate:      false,
			expectedInitialInterval: 10 * time.Second,
			expectedMaxInterval:     60 * time.Second,
			expectedMultiplier:      1.2,
			expectedRandomization:   0.2,
			expectedErrorRegexps:    []string{"timeout", "temporary"},
			expectedStatusCodes:     retry.NewRetryValueNull().GetDefaultRetryableStatusCodes(),
		},
		{
			name: "custom retry with read after create",
			rtry: retry.NewRetryValueMust(retry.NewRetryValueNull().AttributeTypes(ctx), map[string]attr.Value{
				"interval_seconds":     basetypes.NewInt64Value(10),
				"max_interval_seconds": basetypes.NewInt64Value(60),
				"multiplier":           basetypes.NewFloat64Value(1.2),
				"randomization_factor": basetypes.NewFloat64Value(0.2),
				"error_message_regex":  basetypes.NewListValueMust(types.StringType, []attr.Value{basetypes.NewStringValue("timeout"), basetypes.NewStringValue("temporary")}),
			}),
			useReadAfterCreate:      true,
			expectedInitialInterval: 10 * time.Second,
			expectedMaxInterval:     60 * time.Second,
			expectedMultiplier:      1.2,
			expectedRandomization:   0.2,
			expectedStatusCodes:     retry.NewRetryValueNull().GetDefaultRetryableReadAfterCreateStatusCodes(),
			expectedErrorRegexps:    []string{"timeout", "temporary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backOff, errRegExps, statusCodes := configureCustomRetry(ctx, tt.rtry, tt.useReadAfterCreate)

			assert.Equalf(t, tt.expectedInitialInterval, backOff.InitialInterval, "InitialInterval")
			assert.Equalf(t, tt.expectedMaxInterval, backOff.MaxInterval, "MaxInterval")
			assert.Equalf(t, tt.expectedMultiplier, backOff.Multiplier, "Multiplier")
			assert.Equalf(t, tt.expectedRandomization, backOff.RandomizationFactor, "RandomizationFactor")
			assert.Equalf(t, tt.expectedStatusCodes, statusCodes, "StatusCodes")

			actualErrorRegexps := make([]string, len(errRegExps))
			for i, re := range errRegExps {
				actualErrorRegexps[i] = re.String()
			}
			assert.Equal(t, tt.expectedErrorRegexps, actualErrorRegexps, "ErrorRegexps")
		})
	}
}
