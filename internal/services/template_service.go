// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"strings"
	"text/template"
)

// TemplateService defines the interface for template operations.
type TemplateService interface {
	// ProcessTemplate processes a template file and writes the result to the target
	ProcessTemplate(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error

	// RenderTemplate renders a template string with the provided variables
	RenderTemplate(ctx context.Context, templateContent string, variables map[string]interface{}, config *TemplateConfig) (string, error)

	// ValidateTemplate validates template syntax without rendering
	ValidateTemplate(ctx context.Context, templateContent string, engine TemplateEngine) (*ValidationResult, error)

	// GetSupportedEngines returns a list of supported template engines
	GetSupportedEngines() []TemplateEngine

	// CreateEngine creates a template engine with custom functions
	CreateEngine(engine TemplateEngine, functions template.FuncMap) (Engine, error)

	// GetPlatformVariables returns platform-specific template variables
	GetPlatformVariables(ctx context.Context) map[string]interface{}
}

// TemplateEngine represents different template engines.
type TemplateEngine string

const (
	TemplateEngineGo         TemplateEngine = "go"
	TemplateEngineHandlebars TemplateEngine = "handlebars"
	TemplateEngineMustache   TemplateEngine = "mustache"
)

// TemplateConfig contains configuration for template processing.
type TemplateConfig struct {
	// Engine specifies which template engine to use
	Engine TemplateEngine

	// Variables contains template variables
	Variables map[string]interface{}

	// Functions contains custom template functions
	Functions template.FuncMap

	// PlatformVariables includes platform-specific variables
	PlatformVariables bool

	// StrictMode enables strict variable checking
	StrictMode bool

	// Delimiters specifies custom template delimiters
	Delimiters *TemplateDelimiters

	// DryRun indicates if this is a dry run
	DryRun bool
}

// TemplateDelimiters defines custom template delimiters.
type TemplateDelimiters struct {
	Left  string
	Right string
}

// Engine represents a template engine instance.
type Engine interface {
	// Render renders a template with the provided variables
	Render(templateContent string, variables map[string]interface{}) (string, error)

	// Validate validates template syntax
	Validate(templateContent string) error

	// GetEngine returns the engine type
	GetEngine() TemplateEngine
}

// TemplateResult contains the result of template processing.
type TemplateResult struct {
	// Content is the rendered template content
	Content string

	// Variables contains the variables used in rendering
	Variables map[string]interface{}

	// Engine is the template engine used
	Engine TemplateEngine

	// ProcessingTime is how long the rendering took
	ProcessingTime int64
}

// DefaultTemplateService provides a default implementation of TemplateService.
type DefaultTemplateService struct {
	// platformProvider provides platform-specific operations
	platformProvider TemplatePlatformProvider

	// dryRun indicates if operations should be simulated
	dryRun bool

	// engines contains registered template engines
	engines map[TemplateEngine]Engine
}

// TemplatePlatformProvider defines platform-specific operations needed by the template service.
type TemplatePlatformProvider interface {
	// ReadFile reads a file and returns its content
	ReadFile(path string) ([]byte, error)

	// WriteFile writes content to a file
	WriteFile(path string, content []byte, mode uint32) error

	// GetPlatformInfo returns platform-specific information
	GetPlatformInfo() map[string]interface{}

	// ExpandPath expands a path with environment variables
	ExpandPath(path string) (string, error)
}

// NewDefaultTemplateService creates a new default template service.
func NewDefaultTemplateService(platformProvider TemplatePlatformProvider, dryRun bool) *DefaultTemplateService {
	service := &DefaultTemplateService{
		platformProvider: platformProvider,
		dryRun:           dryRun,
		engines:          make(map[TemplateEngine]Engine),
	}

	// Register default engines
	service.registerDefaultEngines()

	return service
}

// ProcessTemplate implements TemplateService.ProcessTemplate.
func (s *DefaultTemplateService) ProcessTemplate(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error {
	if s.dryRun || config.DryRun {
		return s.simulateProcessTemplate(ctx, sourcePath, targetPath, config)
	}

	return s.performProcessTemplate(ctx, sourcePath, targetPath, config)
}

// RenderTemplate implements TemplateService.RenderTemplate.
func (s *DefaultTemplateService) RenderTemplate(ctx context.Context, templateContent string, variables map[string]interface{}, config *TemplateConfig) (string, error) {
	engine, exists := s.engines[config.Engine]
	if !exists {
		return "", &TemplateError{
			Type:    "unsupported_engine",
			Message: "Unsupported template engine: " + string(config.Engine),
			Engine:  config.Engine,
		}
	}

	// Merge platform variables if requested
	if config.PlatformVariables {
		platformVars := s.GetPlatformVariables(ctx)
		variables = s.mergeVariables(variables, platformVars)
	}

	return engine.Render(templateContent, variables)
}

// ValidateTemplate implements TemplateService.ValidateTemplate.
func (s *DefaultTemplateService) ValidateTemplate(ctx context.Context, templateContent string, engine TemplateEngine) (*ValidationResult, error) {
	templateEngine, exists := s.engines[engine]
	if !exists {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{"Unsupported template engine: " + string(engine)},
		}, nil
	}

	err := templateEngine.Validate(templateContent)
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, err
	}

	return &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil
}

// GetSupportedEngines implements TemplateService.GetSupportedEngines.
func (s *DefaultTemplateService) GetSupportedEngines() []TemplateEngine {
	engines := make([]TemplateEngine, 0, len(s.engines))
	for engine := range s.engines {
		engines = append(engines, engine)
	}
	return engines
}

// CreateEngine implements TemplateService.CreateEngine.
func (s *DefaultTemplateService) CreateEngine(engine TemplateEngine, functions template.FuncMap) (Engine, error) {
	switch engine {
	case TemplateEngineGo:
		return NewGoEngine(functions), nil
	case TemplateEngineHandlebars:
		return NewHandlebarsEngine(functions), nil
	case TemplateEngineMustache:
		return NewMustacheEngine(functions), nil
	default:
		return nil, &TemplateError{
			Type:    "unsupported_engine",
			Message: "Unsupported template engine: " + string(engine),
			Engine:  engine,
		}
	}
}

// GetPlatformVariables implements TemplateService.GetPlatformVariables.
func (s *DefaultTemplateService) GetPlatformVariables(ctx context.Context) map[string]interface{} {
	return s.platformProvider.GetPlatformInfo()
}

// Helper methods

func (s *DefaultTemplateService) registerDefaultEngines() {
	// Register Go template engine
	if goEngine, err := s.CreateEngine(TemplateEngineGo, nil); err == nil {
		s.engines[TemplateEngineGo] = goEngine
	}

	// Register Handlebars engine
	if handlebarsEngine, err := s.CreateEngine(TemplateEngineHandlebars, nil); err == nil {
		s.engines[TemplateEngineHandlebars] = handlebarsEngine
	}

	// Register Mustache engine
	if mustacheEngine, err := s.CreateEngine(TemplateEngineMustache, nil); err == nil {
		s.engines[TemplateEngineMustache] = mustacheEngine
	}
}

func (s *DefaultTemplateService) simulateProcessTemplate(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error {
	// Simulate template processing for dry run
	return nil
}

func (s *DefaultTemplateService) performProcessTemplate(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error {
	// Read template content
	content, err := s.platformProvider.ReadFile(sourcePath)
	if err != nil {
		return &TemplateError{
			Type:    "read_error",
			Message: "Failed to read template file: " + err.Error(),
			Path:    sourcePath,
		}
	}

	// Render template
	rendered, err := s.RenderTemplate(ctx, string(content), config.Variables, config)
	if err != nil {
		return err
	}

	// Write rendered content
	return s.platformProvider.WriteFile(targetPath, []byte(rendered), 0644)
}

func (s *DefaultTemplateService) mergeVariables(base, additional map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy base variables
	for k, v := range base {
		merged[k] = v
	}

	// Add additional variables (may overwrite base)
	for k, v := range additional {
		merged[k] = v
	}

	return merged
}

// TemplateError represents template-related errors.
type TemplateError struct {
	Type    string
	Message string
	Path    string
	Engine  TemplateEngine
	Line    int
	Column  int
}

func (e *TemplateError) Error() string {
	if e.Path != "" {
		return e.Message + " (path: " + e.Path + ")"
	}
	return e.Message
}

// Engine implementations

// GoEngine implements the Go template engine.
type GoEngine struct {
	functions template.FuncMap
}

func NewGoEngine(functions template.FuncMap) *GoEngine {
	return &GoEngine{
		functions: functions,
	}
}

func (e *GoEngine) Render(templateContent string, variables map[string]interface{}) (string, error) {
	tmpl := template.New("template")
	if e.functions != nil {
		tmpl = tmpl.Funcs(e.functions)
	}

	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, variables)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *GoEngine) Validate(templateContent string) error {
	tmpl := template.New("validation")
	if e.functions != nil {
		tmpl = tmpl.Funcs(e.functions)
	}

	_, err := tmpl.Parse(templateContent)
	return err
}

func (e *GoEngine) GetEngine() TemplateEngine {
	return TemplateEngineGo
}

// HandlebarsEngine implements the Handlebars template engine.
type HandlebarsEngine struct {
	functions template.FuncMap
}

func NewHandlebarsEngine(functions template.FuncMap) *HandlebarsEngine {
	return &HandlebarsEngine{
		functions: functions,
	}
}

func (e *HandlebarsEngine) Render(templateContent string, variables map[string]interface{}) (string, error) {
	// This would integrate with a Handlebars library
	// For now, this is a placeholder
	return "", nil
}

func (e *HandlebarsEngine) Validate(templateContent string) error {
	// Validate Handlebars syntax
	return nil
}

func (e *HandlebarsEngine) GetEngine() TemplateEngine {
	return TemplateEngineHandlebars
}

// MustacheEngine implements the Mustache template engine.
type MustacheEngine struct {
	functions template.FuncMap
}

func NewMustacheEngine(functions template.FuncMap) *MustacheEngine {
	return &MustacheEngine{
		functions: functions,
	}
}

func (e *MustacheEngine) Render(templateContent string, variables map[string]interface{}) (string, error) {
	// This would integrate with a Mustache library
	// For now, this is a placeholder
	return "", nil
}

func (e *MustacheEngine) Validate(templateContent string) error {
	// Validate Mustache syntax
	return nil
}

func (e *MustacheEngine) GetEngine() TemplateEngine {
	return TemplateEngineMustache
}
