// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"fmt"
	"testing"
	"text/template"
)

// CombinedMockProvider implements both PlatformProvider and TemplatePlatformProvider.
type CombinedMockProvider struct {
	*MockPlatformProvider
	*MockTemplatePlatformProvider
}

func NewCombinedMockProvider() *CombinedMockProvider {
	return &CombinedMockProvider{
		MockPlatformProvider:         &MockPlatformProvider{},
		MockTemplatePlatformProvider: NewMockTemplatePlatformProvider(),
	}
}

func TestNewServiceRegistry(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)
	if registry == nil {
		t.Fatal("Expected service registry to be created")
	}

	if registry.dryRun != false {
		t.Error("Expected dry run to be false")
	}

	// Check that services are initialized
	if registry.backupService == nil {
		t.Error("Expected backup service to be initialized")
	}

	if registry.templateService == nil {
		t.Error("Expected template service to be initialized")
	}
}

func TestServiceRegistry_BackupService(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	backupService := registry.BackupService()
	if backupService == nil {
		t.Fatal("Expected backup service")
	}

	// Test that we get the same instance
	backupService2 := registry.BackupService()
	if backupService != backupService2 {
		t.Error("Expected same backup service instance")
	}
}

func TestServiceRegistry_TemplateService(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	templateService := registry.TemplateService()
	if templateService == nil {
		t.Fatal("Expected template service")
	}

	// Test that we get the same instance
	templateService2 := registry.TemplateService()
	if templateService != templateService2 {
		t.Error("Expected same template service instance")
	}
}

func TestServiceRegistry_SetDryRun(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	// Initial state
	if registry.dryRun {
		t.Error("Expected initial dry run to be false")
	}

	// Change to dry run
	registry.SetDryRun(true)
	if !registry.dryRun {
		t.Error("Expected dry run to be true after SetDryRun(true)")
	}

	// Verify services are updated (if they support dynamic changes)
	// This tests the interface, actual behavior depends on implementation

	// Change back to normal mode
	registry.SetDryRun(false)
	if registry.dryRun {
		t.Error("Expected dry run to be false after SetDryRun(false)")
	}
}

func TestServiceRegistry_Shutdown(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	ctx := context.Background()
	err := registry.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Services should still be accessible after shutdown
	// (current implementation doesn't actually shut down services)
	backupService := registry.BackupService()
	if backupService == nil {
		t.Error("Expected backup service to still be accessible after shutdown")
	}
}

func TestServiceRegistry_HealthCheck(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	ctx := context.Background()
	result := registry.HealthCheck(ctx)

	if result == nil {
		t.Fatal("Expected health check result")
	}

	if !result.Healthy {
		t.Error("Expected overall health to be true")
	}

	if len(result.Services) == 0 {
		t.Error("Expected service health information")
	}

	// Check individual service health
	expectedServices := []string{"backup", "template"}
	for _, serviceName := range expectedServices {
		serviceHealth, exists := result.Services[serviceName]
		if !exists {
			t.Errorf("Expected health info for service %s", serviceName)
			continue
		}

		if !serviceHealth.Healthy {
			t.Errorf("Expected service %s to be healthy", serviceName)
		}

		if serviceHealth.Message == "" {
			t.Errorf("Expected health message for service %s", serviceName)
		}
	}
}

func TestServiceRegistry_HealthCheckWithUnhealthyService(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	// Create a template service that reports no engines (unhealthy)
	mockTemplateService := &MockTemplateService{
		GetSupportedEnginesFunc: func() []TemplateEngine {
			return []TemplateEngine{} // No engines = unhealthy
		},
	}
	registry.templateService = mockTemplateService

	ctx := context.Background()
	result := registry.HealthCheck(ctx)

	if result == nil {
		t.Fatal("Expected health check result")
	}

	if result.Healthy {
		t.Error("Expected overall health to be false when template service is unhealthy")
	}

	templateHealth, exists := result.Services["template"]
	if !exists {
		t.Fatal("Expected template service health info")
	}

	if templateHealth.Healthy {
		t.Error("Expected template service to be unhealthy")
	}
}

func TestServiceRegistry_ConcurrentAccess(t *testing.T) {
	mockProvider := NewCombinedMockProvider()
	config := ServiceConfig{
		DryRun:           false,
		PlatformProvider: mockProvider,
	}

	registry := NewServiceRegistry(config)

	// Test concurrent access to services
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			// Access services concurrently
			backupService := registry.BackupService()
			templateService := registry.TemplateService()

			if backupService == nil {
				results <- fmt.Errorf("backup service is nil")
				return
			}

			if templateService == nil {
				results <- fmt.Errorf("template service is nil")
				return
			}

			// Perform health check
			ctx := context.Background()
			healthResult := registry.HealthCheck(ctx)
			if healthResult == nil {
				results <- fmt.Errorf("health check result is nil")
				return
			}

			// Change dry run mode
			registry.SetDryRun(true)
			registry.SetDryRun(false)

			results <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent access failed: %v", err)
		}
	}
}

// MockTemplateService implements TemplateService for testing.
type MockTemplateService struct {
	ProcessTemplateFunc      func(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error
	RenderTemplateFunc       func(ctx context.Context, templateContent string, variables map[string]interface{}, config *TemplateConfig) (string, error)
	ValidateTemplateFunc     func(ctx context.Context, templateContent string, engine TemplateEngine) (*ValidationResult, error)
	GetSupportedEnginesFunc  func() []TemplateEngine
	CreateEngineFunc         func(engine TemplateEngine, functions template.FuncMap) (Engine, error)
	GetPlatformVariablesFunc func(ctx context.Context) map[string]interface{}
}

func (m *MockTemplateService) ProcessTemplate(ctx context.Context, sourcePath, targetPath string, config *TemplateConfig) error {
	if m.ProcessTemplateFunc != nil {
		return m.ProcessTemplateFunc(ctx, sourcePath, targetPath, config)
	}
	return nil
}

func (m *MockTemplateService) RenderTemplate(ctx context.Context, templateContent string, variables map[string]interface{}, config *TemplateConfig) (string, error) {
	if m.RenderTemplateFunc != nil {
		return m.RenderTemplateFunc(ctx, templateContent, variables, config)
	}
	return "rendered", nil
}

func (m *MockTemplateService) ValidateTemplate(ctx context.Context, templateContent string, engine TemplateEngine) (*ValidationResult, error) {
	if m.ValidateTemplateFunc != nil {
		return m.ValidateTemplateFunc(ctx, templateContent, engine)
	}
	return &ValidationResult{Valid: true}, nil
}

func (m *MockTemplateService) GetSupportedEngines() []TemplateEngine {
	if m.GetSupportedEnginesFunc != nil {
		return m.GetSupportedEnginesFunc()
	}
	return []TemplateEngine{TemplateEngineGo}
}

func (m *MockTemplateService) CreateEngine(engine TemplateEngine, functions template.FuncMap) (Engine, error) {
	if m.CreateEngineFunc != nil {
		return m.CreateEngineFunc(engine, functions)
	}
	return NewGoEngine(functions), nil
}

func (m *MockTemplateService) GetPlatformVariables(ctx context.Context) map[string]interface{} {
	if m.GetPlatformVariablesFunc != nil {
		return m.GetPlatformVariablesFunc(ctx)
	}
	return map[string]interface{}{"platform": "test"}
}

func TestServiceHealth(t *testing.T) {
	health := ServiceHealth{
		Healthy:   true,
		Message:   "Service operational",
		LastCheck: "2023-01-01T00:00:00Z",
	}

	if !health.Healthy {
		t.Error("Expected service to be healthy")
	}

	if health.Message != "Service operational" {
		t.Errorf("Expected message 'Service operational', got '%s'", health.Message)
	}
}

func TestHealthCheckResult(t *testing.T) {
	result := &HealthCheckResult{
		Healthy: true,
		Services: map[string]ServiceHealth{
			"test-service": {
				Healthy: true,
				Message: "OK",
			},
		},
		Message: "All services healthy",
	}

	if !result.Healthy {
		t.Error("Expected result to be healthy")
	}

	if len(result.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(result.Services))
	}

	serviceHealth, exists := result.Services["test-service"]
	if !exists {
		t.Error("Expected test-service in results")
	}

	if !serviceHealth.Healthy {
		t.Error("Expected test-service to be healthy")
	}
}

func TestServiceConfig(t *testing.T) {
	mockProvider := NewCombinedMockProvider()

	config := ServiceConfig{
		DryRun:           true,
		PlatformProvider: mockProvider,
	}

	if !config.DryRun {
		t.Error("Expected dry run to be true")
	}

	if config.PlatformProvider == nil {
		t.Error("Expected platform provider to be set")
	}

	// Test that the provider implements both interfaces
	if _, ok := config.PlatformProvider.(PlatformProvider); !ok {
		t.Error("Expected platform provider to implement PlatformProvider")
	}

	if _, ok := config.PlatformProvider.(TemplatePlatformProvider); !ok {
		t.Error("Expected platform provider to implement TemplatePlatformProvider")
	}
}
