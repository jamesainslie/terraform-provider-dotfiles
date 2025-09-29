// Copyright (c) HashCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package platform

import (
	"context"
	"fmt"
)

// DeclarativeResource defines the interface that all declarative resources must implement.
// This interface ensures that resources follow the declarative paradigm by declaring
// desired state rather than imperative actions.
type DeclarativeResource interface {
	// GetDesiredState returns the desired state as declared in the resource configuration
	GetDesiredState() interface{}

	// GetActualState retrieves the actual current state of the managed resource
	GetActualState(ctx context.Context) (interface{}, error)

	// ApplyState applies changes to make the actual state match the desired state
	ApplyState(ctx context.Context) error

	// ValidateState validates that the desired state is achievable and valid
	ValidateState() error

	// DetectDrift detects if the actual state has drifted from the desired state
	DetectDrift() (bool, error)

	// Description returns a human-readable description of the resource
	Description() string
}

// ResourceState represents the state comparison result for a declarative resource
type ResourceState struct {
	Resource    DeclarativeResource
	Desired     interface{}
	Actual      interface{}
	HasDrift    bool
	NeedsUpdate bool
	LastApplied string
	Error       error
}

// ResourceError provides structured error information for resource operations
type ResourceError struct {
	Type        string                 // Error type (validation, apply, drift, etc.)
	Resource    string                 // Resource identifier
	Operation   string                 // Operation that failed
	Underlying  error                  // Underlying error
	Context     map[string]interface{} // Additional context
	Recoverable bool                   // Whether the error is recoverable
}

func (r *ResourceError) Error() string {
	if len(r.Context) > 0 {
		return fmt.Sprintf("%s error in %s during %s: %v (context: %+v)",
			r.Type, r.Resource, r.Operation, r.Underlying, r.Context)
	}
	return fmt.Sprintf("%s error in %s during %s: %v",
		r.Type, r.Resource, r.Operation, r.Underlying)
}

func (r *ResourceError) Unwrap() error {
	return r.Underlying
}

// NewResourceError creates a new ResourceError with the provided details
func NewResourceError(errType, resource, operation string, underlying error) *ResourceError {
	return &ResourceError{
		Type:        errType,
		Resource:    resource,
		Operation:   operation,
		Underlying:  underlying,
		Context:     make(map[string]interface{}),
		Recoverable: false,
	}
}

// WithContext adds context information to the resource error
func (r *ResourceError) WithContext(key string, value interface{}) *ResourceError {
	if r.Context == nil {
		r.Context = make(map[string]interface{})
	}
	r.Context[key] = value
	return r
}

// WithRecoverable marks the error as recoverable or not
func (r *ResourceError) WithRecoverable(recoverable bool) *ResourceError {
	r.Recoverable = recoverable
	return r
}

// ValidationRule defines a validation rule that can be applied to resources
type ValidationRule interface {
	// Validate performs the validation check on a resource
	Validate(ctx context.Context, resource interface{}) error

	// Description returns a human-readable description of the validation rule
	Description() string
}

// ResourceValidator provides validation capabilities for resources
type ResourceValidator struct {
	Rules []ValidationRule
}

// NewResourceValidator creates a new resource validator with the provided rules
func NewResourceValidator(rules ...ValidationRule) *ResourceValidator {
	return &ResourceValidator{
		Rules: rules,
	}
}

// Validate runs all validation rules against the resource
func (rv *ResourceValidator) Validate(ctx context.Context, resource interface{}) error {
	for _, rule := range rv.Rules {
		if err := rule.Validate(ctx, resource); err != nil {
			return fmt.Errorf("validation rule '%s' failed: %w", rule.Description(), err)
		}
	}
	return nil
}

// AddRule adds a validation rule to the validator
func (rv *ResourceValidator) AddRule(rule ValidationRule) {
	rv.Rules = append(rv.Rules, rule)
}
