// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NotEmptyValidator validates that a string is not empty.
type NotEmptyValidator struct{}

// Description returns a description of the validator.
func (v NotEmptyValidator) Description(_ context.Context) string {
	return "value must not be empty"
}

// MarkdownDescription returns a markdown description of the validator.
func (v NotEmptyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v NotEmptyValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if strings.TrimSpace(value) == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Value",
			"Value cannot be empty or contain only whitespace.",
		)
	}
}

// NotEmpty returns a validator which ensures the configured attribute value is not empty.
func NotEmpty() validator.String {
	return NotEmptyValidator{}
}

// OneOfValidator validates that a string is one of a specified set of values.
type OneOfValidator struct {
	values []string
}

// Description returns a description of the validator.
func (v OneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(v.values, ", "))
}

// MarkdownDescription returns a markdown description of the validator.
func (v OneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v OneOfValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	for _, allowedValue := range v.values {
		if value == allowedValue {
			return // Value is valid
		}
	}

	response.Diagnostics.AddAttributeError(
		request.Path,
		"Invalid Value",
		fmt.Sprintf("Value %q is not valid. Expected one of: %s", value, strings.Join(v.values, ", ")),
	)
}

// OneOf returns a validator which ensures the configured attribute value is one of the specified values.
func OneOf(values ...string) validator.String {
	return OneOfValidator{
		values: values,
	}
}
