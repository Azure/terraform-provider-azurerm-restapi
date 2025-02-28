package retry

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.ObjectTypable  = RetryType{}
	_ basetypes.ObjectValuable = RetryValue{}

	defaultRetryableStatusCodes                = []int{429}
	defaultRetryableReadAfterCreateStatusCodes = []int{404, 403}
)

const (
	DefaultIntervalSeconds     = 10
	DefaultMaxIntervalSeconds  = 180
	DefaultMultiplier          = 1.5
	DefaultRandomizationFactor = 0.5
)

var _ basetypes.ObjectTypable = RetryType{}

type RetryType struct {
	basetypes.ObjectType
}

func (t RetryType) Equal(o attr.Type) bool {
	other, ok := o.(RetryType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t RetryType) String() string {
	return "RetryType"
}

func (t RetryType) ValueFromObject(ctx context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributes := in.Attributes()

	errorMessageRegexAttribute, ok := attributes["error_message_regex"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`error_message_regex is missing from object`)

		return nil, diags
	}

	errorMessageRegexVal, ok := errorMessageRegexAttribute.(basetypes.ListValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`error_message_regex expected to be basetypes.ListValue, was: %T`, errorMessageRegexAttribute))
	}

	intervalSecondsAttribute, ok := attributes["interval_seconds"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`interval_seconds is missing from object`)

		return nil, diags
	}

	intervalSecondsVal, ok := intervalSecondsAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`interval_seconds expected to be basetypes.Int64Value, was: %T`, intervalSecondsAttribute))
	}

	maxIntervalSecondsAttribute, ok := attributes["max_interval_seconds"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`max_interval_seconds is missing from object`)

		return nil, diags
	}

	maxIntervalSecondsVal, ok := maxIntervalSecondsAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`max_interval_seconds expected to be basetypes.Int64Value, was: %T`, maxIntervalSecondsAttribute))
	}

	multiplierAttribute, ok := attributes["multiplier"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`multiplier is missing from object`)

		return nil, diags
	}

	multiplierVal, ok := multiplierAttribute.(basetypes.Float64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`multiplier expected to be basetypes.Float64Value, was: %T`, multiplierAttribute))
	}

	randomizationFactorAttribute, ok := attributes["randomization_factor"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`randomization_factor is missing from object`)

		return nil, diags
	}

	randomizationFactorVal, ok := randomizationFactorAttribute.(basetypes.Float64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`randomization_factor expected to be basetypes.Float64Value, was: %T`, randomizationFactorAttribute))
	}

	if diags.HasError() {
		return nil, diags
	}

	return RetryValue{
		ErrorMessageRegex:   errorMessageRegexVal,
		IntervalSeconds:     intervalSecondsVal,
		MaxIntervalSeconds:  maxIntervalSecondsVal,
		Multiplier:          multiplierVal,
		RandomizationFactor: randomizationFactorVal,
		state:               attr.ValueStateKnown,
	}, diags
}

func NewRetryValueNull() RetryValue {
	return RetryValue{
		state: attr.ValueStateNull,
	}
}

func NewRetryValueUnknown() RetryValue {
	return RetryValue{
		state: attr.ValueStateUnknown,
	}
}

func NewRetryValue(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) (RetryValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/521
	ctx := context.Background()

	for name, attributeType := range attributeTypes {
		attribute, ok := attributes[name]

		if !ok {
			diags.AddError(
				"Missing RetryValue Attribute Value",
				"While creating a RetryValue value, a missing attribute value was detected. "+
					"A RetryValue must contain values for all attributes, even if null or unknown. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("RetryValue Attribute Name (%s) Expected Type: %s", name, attributeType.String()),
			)

			continue
		}

		if !attributeType.Equal(attribute.Type(ctx)) {
			diags.AddError(
				"Invalid RetryValue Attribute Type",
				"While creating a RetryValue value, an invalid attribute value was detected. "+
					"A RetryValue must use a matching attribute type for the value. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("RetryValue Attribute Name (%s) Expected Type: %s\n", name, attributeType.String())+
					fmt.Sprintf("RetryValue Attribute Name (%s) Given Type: %s", name, attribute.Type(ctx)),
			)
		}
	}

	for name := range attributes {
		_, ok := attributeTypes[name]

		if !ok {
			diags.AddError(
				"Extra RetryValue Attribute Value",
				"While creating a RetryValue value, an extra attribute value was detected. "+
					"A RetryValue must not contain values beyond the expected attribute types. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("Extra RetryValue Attribute Name: %s", name),
			)
		}
	}

	if diags.HasError() {
		return NewRetryValueUnknown(), diags
	}

	errorMessageRegexAttribute, ok := attributes["error_message_regex"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`error_message_regex is missing from object`)

		return NewRetryValueUnknown(), diags
	}

	errorMessageRegexVal, ok := errorMessageRegexAttribute.(basetypes.ListValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`error_message_regex expected to be basetypes.ListValue, was: %T`, errorMessageRegexAttribute))
	}

	intervalSecondsAttribute, ok := attributes["interval_seconds"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`interval_seconds is missing from object`)

		return NewRetryValueUnknown(), diags
	}

	intervalSecondsVal, ok := intervalSecondsAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`interval_seconds expected to be basetypes.Int64Value, was: %T`, intervalSecondsAttribute))
	}

	maxIntervalSecondsAttribute, ok := attributes["max_interval_seconds"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`max_interval_seconds is missing from object`)

		return NewRetryValueUnknown(), diags
	}

	maxIntervalSecondsVal, ok := maxIntervalSecondsAttribute.(basetypes.Int64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`max_interval_seconds expected to be basetypes.Int64Value, was: %T`, maxIntervalSecondsAttribute))
	}

	multiplierAttribute, ok := attributes["multiplier"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`multiplier is missing from object`)

		return NewRetryValueUnknown(), diags
	}

	multiplierVal, ok := multiplierAttribute.(basetypes.Float64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`multiplier expected to be basetypes.Float64Value, was: %T`, multiplierAttribute))
	}

	randomizationFactorAttribute, ok := attributes["randomization_factor"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`randomization_factor is missing from object`)

		return NewRetryValueUnknown(), diags
	}

	randomizationFactorVal, ok := randomizationFactorAttribute.(basetypes.Float64Value)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`randomization_factor expected to be basetypes.Float64Value, was: %T`, randomizationFactorAttribute))
	}

	if diags.HasError() {
		return NewRetryValueUnknown(), diags
	}

	return RetryValue{
		ErrorMessageRegex:   errorMessageRegexVal,
		IntervalSeconds:     intervalSecondsVal,
		MaxIntervalSeconds:  maxIntervalSecondsVal,
		Multiplier:          multiplierVal,
		RandomizationFactor: randomizationFactorVal,
		state:               attr.ValueStateKnown,
	}, diags
}

func NewRetryValueMust(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) RetryValue {
	object, diags := NewRetryValue(attributeTypes, attributes)

	if diags.HasError() {
		// This could potentially be added to the diag package.
		diagsStrings := make([]string, 0, len(diags))

		for _, diagnostic := range diags {
			diagsStrings = append(diagsStrings, fmt.Sprintf(
				"%s | %s | %s",
				diagnostic.Severity(),
				diagnostic.Summary(),
				diagnostic.Detail()))
		}

		panic("NewRetryValueMust received error(s): " + strings.Join(diagsStrings, "\n"))
	}

	return object
}

func (t RetryType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if in.Type() == nil {
		return NewRetryValueNull(), nil
	}

	if !in.Type().Equal(t.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", t.TerraformType(ctx), in.Type())
	}

	if !in.IsKnown() {
		return NewRetryValueUnknown(), nil
	}

	if in.IsNull() {
		return NewRetryValueNull(), nil
	}

	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}

	err := in.As(&val)

	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := t.AttrTypes[k].ValueFromTerraform(ctx, v)

		if err != nil {
			return nil, err
		}

		attributes[k] = a
	}

	return NewRetryValueMust(RetryValue{}.AttributeTypes(ctx), attributes), nil
}

func (t RetryType) ValueType(ctx context.Context) attr.Value {
	return RetryValue{}
}

var _ basetypes.ObjectValuable = RetryValue{}

type RetryValue struct {
	ErrorMessageRegex   basetypes.ListValue    `tfsdk:"error_message_regex"`
	IntervalSeconds     basetypes.Int64Value   `tfsdk:"interval_seconds"`
	MaxIntervalSeconds  basetypes.Int64Value   `tfsdk:"max_interval_seconds"`
	Multiplier          basetypes.Float64Value `tfsdk:"multiplier"`
	RandomizationFactor basetypes.Float64Value `tfsdk:"randomization_factor"`
	state               attr.ValueState
}

func (v RetryValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	attrTypes := make(map[string]tftypes.Type, 5)

	var val tftypes.Value
	var err error

	attrTypes["error_message_regex"] = basetypes.ListType{
		ElemType: types.StringType,
	}.TerraformType(ctx)
	attrTypes["interval_seconds"] = basetypes.Int64Type{}.TerraformType(ctx)
	attrTypes["max_interval_seconds"] = basetypes.Int64Type{}.TerraformType(ctx)
	attrTypes["multiplier"] = basetypes.Float64Type{}.TerraformType(ctx)
	attrTypes["randomization_factor"] = basetypes.Float64Type{}.TerraformType(ctx)

	objectType := tftypes.Object{AttributeTypes: attrTypes}

	switch v.state {
	case attr.ValueStateKnown:
		vals := make(map[string]tftypes.Value, 5)

		val, err = v.ErrorMessageRegex.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["error_message_regex"] = val

		val, err = v.IntervalSeconds.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["interval_seconds"] = val

		val, err = v.MaxIntervalSeconds.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["max_interval_seconds"] = val

		val, err = v.Multiplier.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["multiplier"] = val

		val, err = v.RandomizationFactor.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["randomization_factor"] = val

		if err := tftypes.ValidateValue(objectType, vals); err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		return tftypes.NewValue(objectType, vals), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(objectType, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(objectType, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled Object state in ToTerraformValue: %s", v.state))
	}
}

func (v RetryValue) IsNull() bool {
	return v.state == attr.ValueStateNull
}

func (v RetryValue) IsUnknown() bool {
	return v.state == attr.ValueStateUnknown
}

func (v RetryValue) String() string {
	return "RetryValue"
}

func (v RetryValue) ToObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	var errorMessageRegexVal basetypes.ListValue
	switch {
	case v.ErrorMessageRegex.IsUnknown():
		errorMessageRegexVal = types.ListUnknown(types.StringType)
	case v.ErrorMessageRegex.IsNull():
		errorMessageRegexVal = types.ListNull(types.StringType)
	default:
		var d diag.Diagnostics
		errorMessageRegexVal, d = types.ListValue(types.StringType, v.ErrorMessageRegex.Elements())
		diags.Append(d...)
	}

	if diags.HasError() {
		return types.ObjectUnknown(map[string]attr.Type{
			"error_message_regex": basetypes.ListType{
				ElemType: types.StringType,
			},
			"interval_seconds":     basetypes.Int64Type{},
			"max_interval_seconds": basetypes.Int64Type{},
			"multiplier":           basetypes.Float64Type{},
			"randomization_factor": basetypes.Float64Type{},
		}), diags
	}

	attributeTypes := map[string]attr.Type{
		"error_message_regex": basetypes.ListType{
			ElemType: types.StringType,
		},
		"interval_seconds":     basetypes.Int64Type{},
		"max_interval_seconds": basetypes.Int64Type{},
		"multiplier":           basetypes.Float64Type{},
		"randomization_factor": basetypes.Float64Type{},
	}

	if v.IsNull() {
		return types.ObjectNull(attributeTypes), diags
	}

	if v.IsUnknown() {
		return types.ObjectUnknown(attributeTypes), diags
	}

	objVal, diags := types.ObjectValue(
		attributeTypes,
		map[string]attr.Value{
			"error_message_regex":  errorMessageRegexVal,
			"interval_seconds":     v.IntervalSeconds,
			"max_interval_seconds": v.MaxIntervalSeconds,
			"multiplier":           v.Multiplier,
			"randomization_factor": v.RandomizationFactor,
		})

	return objVal, diags
}

func (v RetryValue) Equal(o attr.Value) bool {
	other, ok := o.(RetryValue)

	if !ok {
		return false
	}

	if v.state != other.state {
		return false
	}

	if v.state != attr.ValueStateKnown {
		return true
	}

	if !v.ErrorMessageRegex.Equal(other.ErrorMessageRegex) {
		return false
	}

	if !v.IntervalSeconds.Equal(other.IntervalSeconds) {
		return false
	}

	if !v.MaxIntervalSeconds.Equal(other.MaxIntervalSeconds) {
		return false
	}

	if !v.Multiplier.Equal(other.Multiplier) {
		return false
	}

	if !v.RandomizationFactor.Equal(other.RandomizationFactor) {
		return false
	}

	return true
}

func (v RetryValue) Type(ctx context.Context) attr.Type {
	return RetryType{
		basetypes.ObjectType{
			AttrTypes: v.AttributeTypes(ctx),
		},
	}
}

func (v RetryValue) AttributeTypes(ctx context.Context) map[string]attr.Type {
	return map[string]attr.Type{
		"error_message_regex": basetypes.ListType{
			ElemType: types.StringType,
		},
		"interval_seconds":     basetypes.Int64Type{},
		"max_interval_seconds": basetypes.Int64Type{},
		"multiplier":           basetypes.Float64Type{},
		"randomization_factor": basetypes.Float64Type{},
	}
}

func (v RetryValue) GetIntervalSeconds() int {
	return int(v.IntervalSeconds.ValueInt64())
}

func (v RetryValue) GetIntervalSecondsAsDuration() time.Duration {
	return time.Duration(v.IntervalSeconds.ValueInt64()) * time.Second
}

func (v RetryValue) GetMaxIntervalSeconds() int {
	return int(v.MaxIntervalSeconds.ValueInt64())
}

func (v RetryValue) GetMaxIntervalSecondsAsDuration() time.Duration {
	return time.Duration(v.MaxIntervalSeconds.ValueInt64()) * time.Second
}

func (v RetryValue) GetMultiplier() float64 {
	return v.Multiplier.ValueFloat64()
}

func (v RetryValue) GetRandomizationFactor() float64 {
	return v.RandomizationFactor.ValueFloat64()
}

func (v RetryValue) GetDefaultRetryableStatusCodes() []int {
	return defaultRetryableStatusCodes
}

func (v RetryValue) GetDefaultRetryableReadAfterCreateStatusCodes() []int {
	return defaultRetryableReadAfterCreateStatusCodes
}

func (v RetryValue) GetErrorMessages() []string {
	if v.IsNull() {
		return nil
	}
	if v.IsUnknown() {
		return nil
	}
	res := make([]string, len(v.ErrorMessageRegex.Elements()))
	for i, elem := range v.ErrorMessageRegex.Elements() {
		res[i] = elem.(types.String).ValueString()
	}
	return res
}

func (v RetryValue) GetErrorMessagesRegex() []regexp.Regexp {
	msgs := v.GetErrorMessages()
	if msgs == nil {
		return nil
	}
	res := make([]regexp.Regexp, len(msgs))
	for i, msg := range msgs {
		res[i] = *regexp.MustCompile(msg)
	}
	return res
}
