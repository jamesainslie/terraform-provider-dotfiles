// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"sync"
)

// ServiceRegistry manages all services used by the provider.
type ServiceRegistry struct {
	// Services
	backupService   BackupService
	templateService TemplateService

	// Configuration
	dryRun bool

	// Synchronization
	mu sync.RWMutex
}

// ServiceConfig contains configuration for service initialization.
type ServiceConfig struct {
	// DryRun indicates if services should operate in dry-run mode
	DryRun bool

	// PlatformProvider provides platform-specific operations
	PlatformProvider ServicePlatformProvider
}

// ServicePlatformProvider combines all platform provider interfaces needed by services.
type ServicePlatformProvider interface {
	PlatformProvider
	TemplatePlatformProvider
}

// NewServiceRegistry creates a new service registry with the provided configuration.
func NewServiceRegistry(config ServiceConfig) *ServiceRegistry {
	registry := &ServiceRegistry{
		dryRun: config.DryRun,
	}

	// Initialize services
	registry.backupService = NewDefaultBackupService(config.PlatformProvider, config.DryRun)
	registry.templateService = NewDefaultTemplateService(config.PlatformProvider, config.DryRun)

	return registry
}

// BackupService returns the backup service instance.
func (r *ServiceRegistry) BackupService() BackupService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backupService
}

// TemplateService returns the template service instance.
func (r *ServiceRegistry) TemplateService() TemplateService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.templateService
}

// SetDryRun updates the dry-run mode for all services.
func (r *ServiceRegistry) SetDryRun(dryRun bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dryRun = dryRun

	// Update services if they support dynamic dry-run changes
	if defaultBackup, ok := r.backupService.(*DefaultBackupService); ok {
		defaultBackup.dryRun = dryRun
	}

	if defaultTemplate, ok := r.templateService.(*DefaultTemplateService); ok {
		defaultTemplate.dryRun = dryRun
	}
}

// Shutdown gracefully shuts down all services.
func (r *ServiceRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Services don't currently need shutdown logic, but this provides
	// a hook for future cleanup operations

	return nil
}

// HealthCheck performs a health check on all services.
func (r *ServiceRegistry) HealthCheck(ctx context.Context) *HealthCheckResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := &HealthCheckResult{
		Healthy:  true,
		Services: make(map[string]ServiceHealth),
	}

	// Check backup service
	backupHealth := r.checkBackupService(ctx)
	result.Services["backup"] = backupHealth
	if !backupHealth.Healthy {
		result.Healthy = false
	}

	// Check template service
	templateHealth := r.checkTemplateService(ctx)
	result.Services["template"] = templateHealth
	if !templateHealth.Healthy {
		result.Healthy = false
	}

	return result
}

// Helper methods for health checks

func (r *ServiceRegistry) checkBackupService(_ context.Context) ServiceHealth {
	// Basic health check for backup service
	// In a real implementation, this might test basic operations
	return ServiceHealth{
		Healthy: true,
		Message: "Backup service operational",
	}
}

func (r *ServiceRegistry) checkTemplateService(_ context.Context) ServiceHealth {
	// Basic health check for template service
	engines := r.templateService.GetSupportedEngines()
	if len(engines) == 0 {
		return ServiceHealth{
			Healthy: false,
			Message: "No template engines available",
		}
	}

	return ServiceHealth{
		Healthy: true,
		Message: "Template service operational with " + string(rune(len(engines))) + " engines",
	}
}

// HealthCheckResult contains the result of a service registry health check.
type HealthCheckResult struct {
	// Healthy indicates if all services are healthy
	Healthy bool

	// Services contains health status for individual services
	Services map[string]ServiceHealth

	// Message contains an overall health message
	Message string
}

// ServiceHealth contains health information for a single service.
type ServiceHealth struct {
	// Healthy indicates if the service is healthy
	Healthy bool

	// Message contains a health status message
	Message string

	// LastCheck is when the health check was performed
	LastCheck string
}
