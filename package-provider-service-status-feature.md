# Feature Specification: Package Provider Service Status Extension

> **Provider**: `jamesainslie/package`  
> **Feature**: Service Status Data Sources and Management  
> **Version**: 0.3.0 (Breaking Changes)  
> **Priority**: HIGH - Critical for Infrastructure as Code service management  
> **Created**: September 22, 2025

---

## 🎯 **Executive Summary**

Extend the `jamesainslie/package` provider with native service status checking and management capabilities. This enhancement transforms the provider from a pure package manager into a comprehensive **Package + Service Lifecycle Manager**, enabling complete infrastructure management for development environments.

### **Value Proposition**
- **Infrastructure as Code**: Declarative service management alongside package installation
- **Dependency Management**: Ensure services are running before dependent configurations
- **Cross-Platform**: Unified service management across macOS, Linux, and Windows
- **Integration Ready**: Native Terraform data types for conditional resource creation

---

## 📊 **Current State Analysis**

### **Package Provider Current Capabilities**
```hcl
# Current jamesainslie/package provider functionality
resource "pkg_package" "brew" {
  for_each = var.packages
  
  name     = each.key
  state    = "present"
  manager  = "brew"
  category = each.value.category
}

# ✅ Strengths:
# - Excellent package management across multiple managers
# - Cross-platform support (brew, apt, yum, pacman)
# - Comprehensive package state tracking
# - Good integration with terraform-devenv

# ❌ Gaps:
# - No service status awareness
# - Cannot determine if installed packages provide running services
# - No dependency validation between packages and services
# - Manual service management required outside Terraform
```

### **Current Service Management Approach**
```hcl
# Current workaround in terraform-devenv (suboptimal)
resource "null_resource" "health_check" {
  provisioner "local-exec" {
    command = "check if colima is running..."
  }
  triggers = {
    always_run = timestamp()  # Runs every time!
  }
}

# Issues with current approach:
# ❌ Not idempotent (always_run trigger)
# ❌ No structured data output
# ❌ Cannot use results in terraform logic
# ❌ Platform-specific shell commands
# ❌ Hard to test and maintain
```

---

## 🎯 **Feature Requirements**

### **Primary Requirements**

#### **R1: Service Status Data Sources**
- **R1.1**: Read current status of system services (running/stopped/unknown)
- **R1.2**: Detect service version and health information
- **R1.3**: Cross-platform service detection (macOS/Linux/Windows)
- **R1.4**: Support multiple service managers (launchd, systemd, Windows Services)

#### **R2: Package-Service Relationship Mapping**
- **R2.1**: Automatic detection of services provided by packages
- **R2.2**: Dependency validation (package installed → service available)
- **R2.3**: Service startup requirement checking
- **R2.4**: Package-to-service name mapping database

#### **R3: Terraform Integration**
- **R3.1**: Native terraform data types for service status
- **R3.2**: Conditional resource creation based on service status
- **R3.3**: Service health as terraform state
- **R3.4**: Integration with existing pkg_package resources

### **Secondary Requirements**

#### **R4: Service Management (Optional)**
- **R4.1**: Start/stop service resources (if permitted)
- **R4.2**: Service configuration validation
- **R4.3**: Service restart on configuration changes
- **R4.4**: Graceful service dependency handling

#### **R5: Advanced Features**
- **R5.1**: Service performance metrics
- **R5.2**: Custom health check definitions
- **R5.3**: Service clustering support
- **R5.4**: Integration with monitoring systems

---

## 🏗️ **Proposed Architecture**

### **Data Sources**

#### **`pkg_service_status` (Primary)**
```hcl
# Single service status check
data "pkg_service_status" "colima" {
  name = "colima"
  
  # Optional: Package relationship validation
  required_package = "colima"
  package_manager  = "brew"
  
  # Optional: Custom health check
  health_check = {
    command = "colima status"
    timeout = "5s"
  }
}

# Output structure
output "colima_status" {
  value = {
    running      = data.pkg_service_status.colima.running
    healthy      = data.pkg_service_status.colima.healthy
    version      = data.pkg_service_status.colima.version
    process_id   = data.pkg_service_status.colima.process_id
    start_time   = data.pkg_service_status.colima.start_time
    manager_type = data.pkg_service_status.colima.manager_type
  }
}
```

#### **`pkg_services_overview` (Bulk)**
```hcl
# Multiple services at once (more efficient)
data "pkg_services_overview" "development" {
  services = ["colima", "docker", "postgres", "redis"]
  
  # Optional: Filter by package manager
  package_manager = "brew"
  
  # Optional: Include only services from installed packages
  installed_packages_only = true
}

# Usage in conditional logic
locals {
  container_services_ready = (
    data.pkg_services_overview.development.services.colima.running &&
    data.pkg_services_overview.development.services.docker.healthy
  )
}

resource "dotfiles_file" "docker_config" {
  count = local.container_services_ready ? 1 : 0
  # ... docker configuration
}
```

### **Resources (Optional - Phase 2)**

#### **`pkg_service` (Advanced)**
```hcl
# Service lifecycle management (optional)
resource "pkg_service" "colima" {
  name            = "colima"
  required_package = "colima"
  
  state           = "running"  # running, stopped, enabled, disabled
  restart_policy  = "on_change"
  
  # Configuration
  config = {
    cpu    = 4
    memory = 8192
    disk   = 60
  }
  
  # Dependencies
  depends_on = [pkg_package.brew["colima"]]
}
```

---

## 🔧 **Technical Implementation**

### **Core Interface Design**

#### **Service Detection Interface**
```go
// internal/services/detector.go
type ServiceDetector interface {
    // Core detection
    IsRunning(ctx context.Context, serviceName string) (bool, error)
    GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error)
    
    // Health checking
    CheckHealth(ctx context.Context, serviceName string, healthConfig *HealthConfig) (*HealthResult, error)
    
    // Package relationship
    GetServicesForPackage(packageName string) ([]string, error)
    GetPackageForService(serviceName string) (string, error)
}

// Platform-specific implementations
type MacOSServiceDetector struct{}  // launchd, brew services
type LinuxServiceDetector struct{}  // systemd, init.d
type WindowsServiceDetector struct{} // Windows Services
```

#### **Service Information Model**
```go
type ServiceInfo struct {
    Name         string            `json:"name"`
    Running      bool              `json:"running"`
    Healthy      bool              `json:"healthy"`
    Version      string            `json:"version"`
    ProcessID    int               `json:"process_id"`
    StartTime    time.Time         `json:"start_time"`
    ManagerType  string            `json:"manager_type"` // launchd, systemd, etc.
    Package      *PackageInfo      `json:"package,omitempty"`
    Ports        []int             `json:"ports,omitempty"`
    Metadata     map[string]string `json:"metadata"`
}

type PackageInfo struct {
    Name    string `json:"name"`
    Manager string `json:"manager"`
    Version string `json:"version"`
}
```

### **Platform Detection Matrix**

#### **macOS (launchd + brew services)**
```go
func (m *MacOSServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
    // 1. Check launchctl
    if running, err := m.checkLaunchctl(serviceName); err == nil {
        return running, nil
    }
    
    // 2. Check brew services
    if running, err := m.checkBrewServices(serviceName); err == nil {
        return running, nil
    }
    
    // 3. Check process name
    return m.checkProcessName(serviceName)
}

func (m *MacOSServiceDetector) checkBrewServices(serviceName string) (bool, error) {
    cmd := exec.Command("brew", "services", "list", "--json")
    output, err := cmd.Output()
    if err != nil {
        return false, err
    }
    
    var services []BrewService
    if err := json.Unmarshal(output, &services); err != nil {
        return false, err
    }
    
    for _, service := range services {
        if service.Name == serviceName {
            return service.Status == "started", nil
        }
    }
    
    return false, nil
}
```

#### **Linux (systemd)**
```go
func (l *LinuxServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
    // 1. Check systemctl
    cmd := exec.Command("systemctl", "is-active", serviceName)
    err := cmd.Run()
    if err == nil {
        return true, nil
    }
    
    // 2. Check process name
    return l.checkProcessName(serviceName)
}
```

#### **Windows (Windows Services)**
```go
func (w *WindowsServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
    // Use PowerShell Get-Service
    cmd := exec.Command("powershell", "-Command", 
        fmt.Sprintf("Get-Service -Name %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Status", serviceName))
    output, err := cmd.Output()
    if err != nil {
        return false, err
    }
    
    status := strings.TrimSpace(string(output))
    return status == "Running", nil
}
```

---

## 📋 **Data Source Schema Design**

### **`pkg_service_status` Schema**
```go
func (d *ServiceStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Checks the status of a system service",
        Attributes: map[string]schema.Attribute{
            // Required
            "name": schema.StringAttribute{
                Required: true,
                MarkdownDescription: "Service name to check",
            },
            
            // Optional configuration
            "required_package": schema.StringAttribute{
                Optional: true,
                MarkdownDescription: "Package that provides this service (validates installation)",
            },
            "package_manager": schema.StringAttribute{
                Optional: true,
                Default: stringdefault.StaticString("brew"),
                MarkdownDescription: "Package manager to check for service package",
            },
            "timeout": schema.StringAttribute{
                Optional: true,
                Default: stringdefault.StaticString("10s"),
                MarkdownDescription: "Timeout for service status checks",
            },
            
            // Computed outputs
            "id": schema.StringAttribute{
                Computed: true,
                MarkdownDescription: "Data source identifier",
            },
            "running": schema.BoolAttribute{
                Computed: true,
                MarkdownDescription: "Whether the service is currently running",
            },
            "healthy": schema.BoolAttribute{
                Computed: true,
                MarkdownDescription: "Whether the service passes health checks",
            },
            "version": schema.StringAttribute{
                Computed: true,
                MarkdownDescription: "Service version (if detectable)",
            },
            "process_id": schema.StringAttribute{
                Computed: true,
                MarkdownDescription: "Process ID of the running service",
            },
            "start_time": schema.StringAttribute{
                Computed: true,
                MarkdownDescription: "When the service was started (RFC3339)",
            },
            "manager_type": schema.StringAttribute{
                Computed: true,
                MarkdownDescription: "Service manager type (launchd, systemd, etc.)",
            },
            "ports": schema.ListAttribute{
                Computed: true,
                ElementType: types.Int64Type,
                MarkdownDescription: "Ports the service is listening on",
            },
        },
        Blocks: map[string]schema.Block{
            "health_check": schema.SingleNestedBlock{
                MarkdownDescription: "Custom health check configuration",
                Attributes: map[string]schema.Attribute{
                    "command": schema.StringAttribute{
                        Optional: true,
                        MarkdownDescription: "Custom health check command",
                    },
                    "http_endpoint": schema.StringAttribute{
                        Optional: true,
                        MarkdownDescription: "HTTP endpoint for health checks",
                    },
                    "expected_status": schema.Int64Attribute{
                        Optional: true,
                        Default: int64default.StaticInt64(200),
                        MarkdownDescription: "Expected HTTP status code",
                    },
                    "timeout": schema.StringAttribute{
                        Optional: true,
                        Default: stringdefault.StaticString("5s"),
                        MarkdownDescription: "Health check timeout",
                    },
                },
            },
        },
    }
}
```

---

## 🎯 **Use Cases and Examples**

### **Use Case 1: Container Development Stack**
```hcl
# Package installation with service status validation
module "container_stack" {
  source = "./modules/container-stack"
  
  # Install packages
  packages = {
    colima = { category = "containers" }
    docker = { category = "containers" }
  }
}

# Check service status
data "pkg_service_status" "colima" {
  name             = "colima"
  required_package = "colima"
  package_manager  = "brew"
  
  health_check {
    command = "colima status"
    timeout = "10s"
  }
  
  depends_on = [module.container_stack.pkg_package.brew["colima"]]
}

data "pkg_service_status" "docker" {
  name             = "docker"
  required_package = "docker"
  
  health_check {
    http_endpoint   = "http://localhost:2375/_ping"
    expected_status = 200
    timeout         = "5s"
  }
  
  depends_on = [
    module.container_stack.pkg_package.brew["docker"],
    data.pkg_service_status.colima  # Docker requires Colima on macOS
  ]
}

# Conditional configuration based on service status
resource "dotfiles_file" "docker_config" {
  count = data.pkg_service_status.docker.running && data.pkg_service_status.docker.healthy ? 1 : 0
  
  name        = "docker-daemon-config"
  source_path = "docker/daemon.json"
  target_path = "~/.docker/daemon.json"
  
  # Only configure Docker if it's running and healthy
}

# Service dependency validation
locals {
  container_stack_ready = alltrue([
    data.pkg_service_status.colima.running,
    data.pkg_service_status.docker.running,
    data.pkg_service_status.docker.healthy
  ])
}

output "container_readiness" {
  value = {
    stack_ready = local.container_stack_ready
    services = {
      colima = {
        running = data.pkg_service_status.colima.running
        version = data.pkg_service_status.colima.version
      }
      docker = {
        running = data.pkg_service_status.docker.running
        healthy = data.pkg_service_status.docker.healthy
        version = data.pkg_service_status.docker.version
      }
    }
  }
}
```

### **Use Case 2: Database Development Environment**
```hcl
# Database services with dependency management
data "pkg_services_overview" "databases" {
  services = ["postgresql", "redis", "elasticsearch"]
  
  # Only check services from installed packages
  installed_packages_only = true
  package_manager         = "brew"
}

# Conditional database configuration
resource "dotfiles_file" "postgres_config" {
  count = data.pkg_services_overview.databases.services.postgresql.running ? 1 : 0
  
  name        = "postgresql-config"
  source_path = "databases/postgresql.conf"
  target_path = "~/Library/Application Support/Postgres/postgresql.conf"
}

# Application configuration that depends on services
resource "dotfiles_file" "app_database_config" {
  count = alltrue([
    data.pkg_services_overview.databases.services.postgresql.running,
    data.pkg_services_overview.databases.services.redis.running
  ]) ? 1 : 0
  
  name        = "application-database-config"
  source_path = "app/database.yml"
  target_path = "~/app/config/database.yml"
  is_template = true
  
  template_vars = {
    postgres_host = "localhost"
    postgres_port = "5432"
    redis_host    = "localhost"
    redis_port    = "6379"
    # Template can be conditional based on service status
  }
}
```

### **Use Case 3: Development Tools Ecosystem**
```hcl
# Complete development environment with service awareness
data "pkg_services_overview" "development" {
  services = [
    "colima",      # Container runtime
    "docker",      # Container engine
    "postgres",    # Database
    "redis",       # Cache
    "nginx",       # Web server
    "node",        # Runtime (if applicable)
  ]
  
  include_metrics = true
  health_checks   = true
}

# Generate service status report
output "development_environment_status" {
  value = {
    total_services   = length(data.pkg_services_overview.development.services)
    running_services = length([
      for name, service in data.pkg_services_overview.development.services : 
      name if service.running
    ])
    healthy_services = length([
      for name, service in data.pkg_services_overview.development.services : 
      name if service.running && service.healthy
    ])
    service_details = data.pkg_services_overview.development.services
  }
}

# Environment readiness check
locals {
  environment_ready = alltrue([
    for name, service in data.pkg_services_overview.development.services :
    service.running && service.healthy
  ])
}
```

---

## 🛠️ **Implementation Plan**

### **Phase 1: Core Service Status (1-2 weeks)**

#### **Week 1: Foundation**
```go
// 1. Service detection interface
type ServiceDetector interface {
    IsRunning(ctx context.Context, serviceName string) (bool, error)
    GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error)
    GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error)
}

// 2. Platform-specific implementations
// - MacOSServiceDetector (launchd + brew services)
// - LinuxServiceDetector (systemd + process checks)
// - WindowsServiceDetector (Windows Services API)

// 3. Package-service mapping database
var PackageServiceMap = map[string][]string{
    "colima":      {"colima"},
    "docker":      {"docker", "docker-desktop"},
    "postgresql":  {"postgres", "postgresql"},
    "nginx":       {"nginx"},
    "redis":       {"redis-server", "redis"},
    // ... extensive mapping
}
```

#### **Week 2: Data Sources**
```go
// 1. pkg_service_status data source
type ServiceStatusDataSource struct {
    client   *PackageClient
    detector ServiceDetector
}

// 2. pkg_services_overview data source  
type ServicesOverviewDataSource struct {
    client   *PackageClient
    detector ServiceDetector
}

// 3. Integration with existing package provider
// - Add service status to pkg_package outputs
// - Validate package-service relationships
// - Cross-reference installed packages with running services
```

### **Phase 2: Advanced Features (1-2 weeks)**

#### **Health Checking**
```go
type HealthChecker interface {
    CheckCommand(ctx context.Context, command string, timeout time.Duration) (*HealthResult, error)
    CheckHTTP(ctx context.Context, endpoint string, expectedStatus int, timeout time.Duration) (*HealthResult, error)
    CheckTCP(ctx context.Context, host string, port int, timeout time.Duration) (*HealthResult, error)
}

type HealthResult struct {
    Healthy      bool          `json:"healthy"`
    ResponseTime time.Duration `json:"response_time"`
    Error        string        `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata"`
}
```

#### **Service Management Resources (Optional)**
```go
// Optional: Service lifecycle management
type ServiceResource struct {
    client   *PackageClient
    detector ServiceDetector
    manager  ServiceManager
}

type ServiceManager interface {
    StartService(ctx context.Context, serviceName string) error
    StopService(ctx context.Context, serviceName string) error
    RestartService(ctx context.Context, serviceName string) error
    ConfigureService(ctx context.Context, serviceName string, config map[string]interface{}) error
}
```

---

## 📊 **Service Detection Matrix**

### **Package-to-Service Mapping**

| Package | Service Names | Manager | Detection Method | Health Check |
|---------|---------------|---------|------------------|--------------|
| `colima` | `colima` | brew | `colima status` | Command + Port |
| `docker` | `docker`, `docker-desktop` | brew | `docker info` | HTTP /_ping |
| `postgresql` | `postgres`, `postgresql` | brew | `brew services` + `pg_isready` | TCP 5432 |
| `redis` | `redis-server`, `redis` | brew | `brew services` + `redis-cli` | TCP 6379 |
| `nginx` | `nginx` | brew/apt | `nginx -t` | HTTP 80/443 |
| `mysql` | `mysqld`, `mysql` | brew/apt | `mysqladmin ping` | TCP 3306 |
| `elasticsearch` | `elasticsearch` | brew | HTTP | HTTP 9200/_health |

### **Platform Service Managers**

| Platform | Primary Manager | Secondary | Detection Commands |
|----------|----------------|-----------|-------------------|
| **macOS** | `launchd` | `brew services` | `launchctl list`, `brew services list` |
| **Linux** | `systemd` | `init.d` | `systemctl status`, `service --status-all` |
| **Windows** | `Windows Services` | - | `Get-Service`, `sc query` |

---

## 🧪 **Testing Strategy**

### **Unit Tests**
```go
func TestServiceDetection(t *testing.T) {
    tests := []struct {
        name        string
        platform    string
        serviceName string
        mockOutput  string
        expected    *ServiceInfo
    }{
        {
            name:        "macOS colima running",
            platform:    "darwin",
            serviceName: "colima",
            mockOutput:  `{"name": "colima", "status": "started"}`,
            expected: &ServiceInfo{
                Name:        "colima",
                Running:     true,
                ManagerType: "brew",
            },
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            detector := NewMockServiceDetector(tt.platform, tt.mockOutput)
            result, err := detector.IsRunning(context.Background(), tt.serviceName)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected.Running, result)
        })
    }
}
```

### **Integration Tests**
```go
func TestServiceDataSourceIntegration(t *testing.T) {
    // Test with real services (requires actual services running)
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup: Install package and start service
    // Test: Data source detects service correctly
    // Cleanup: Stop service and verify detection
}
```

### **Acceptance Tests**
```hcl
# Test complete workflow
resource "pkg_package" "test_service" {
  name    = "nginx"
  state   = "present"
  manager = "brew"
}

data "pkg_service_status" "test_nginx" {
  name             = "nginx"
  required_package = "nginx"
  
  health_check {
    http_endpoint   = "http://localhost:80"
    expected_status = 200
  }
  
  depends_on = [pkg_package.test_service]
}

# Verify service detection works
output "test_result" {
  value = {
    package_installed = pkg_package.test_service.state == "present"
    service_detected  = data.pkg_service_status.test_nginx.running
    health_passing    = data.pkg_service_status.test_nginx.healthy
  }
}
```

---

## 🔄 **Migration and Compatibility**

### **Backward Compatibility**
- ✅ **Existing `pkg_package` resources unchanged**
- ✅ **Current package management functionality preserved**
- ✅ **Additive changes only** - no breaking changes to existing resources
- ✅ **Optional service features** - providers work without service data sources

### **Migration Path from Current Approach**
```hcl
# Before: Using null_resource (suboptimal)
resource "null_resource" "service_check" {
  provisioner "local-exec" {
    command = "check_service.sh colima"
  }
  triggers = {
    always_run = timestamp()
  }
}

# After: Using pkg_service_status (terraform-native)
data "pkg_service_status" "colima" {
  name             = "colima"
  required_package = "colima"
}

locals {
  service_ready = data.pkg_service_status.colima.running && 
                  data.pkg_service_status.colima.healthy
}

# Migration benefits:
# ✅ Idempotent (no more always_run triggers)
# ✅ Structured data output (can use in terraform logic)
# ✅ Better error handling and diagnostics
# ✅ Cross-platform compatibility
# ✅ Faster execution (no shell processes)
```

---

## 📈 **Performance and Scalability**

### **Performance Optimizations**
```go
// 1. Efficient bulk service checking
func (d *ServicesOverviewDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // Single command for multiple services instead of individual calls
    services, err := d.detector.GetAllServices(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Service Detection Failed", err.Error())
        return
    }
    
    // Filter to requested services only
    // ... populate response
}

// 2. Caching for repeated calls
type CachedServiceDetector struct {
    underlying ServiceDetector
    cache      map[string]*CachedServiceInfo
    cacheTTL   time.Duration
}

// 3. Parallel health checks
func (h *HealthChecker) CheckMultiple(ctx context.Context, checks []HealthCheck) map[string]*HealthResult {
    results := make(map[string]*HealthResult)
    var wg sync.WaitGroup
    
    for _, check := range checks {
        wg.Add(1)
        go func(c HealthCheck) {
            defer wg.Done()
            result, _ := h.CheckCommand(ctx, c.Command, c.Timeout)
            results[c.ServiceName] = result
        }(check)
    }
    
    wg.Wait()
    return results
}
```

### **Scalability Considerations**
- **Bulk Operations**: Check multiple services in single call
- **Intelligent Caching**: Cache service status with configurable TTL
- **Lazy Loading**: Only check services that are actually referenced
- **Parallel Execution**: Health checks run concurrently
- **Resource Limits**: Configurable timeouts and connection limits

---

## 🔒 **Security and Reliability**

### **Security Measures**
```go
// 1. Command validation
func validateServiceCommand(command string) error {
    // Whitelist allowed commands
    allowedCommands := []string{
        "systemctl", "launchctl", "brew", "docker", "pg_isready",
        "redis-cli", "nginx", "curl", "wget"
    }
    
    cmd := strings.Fields(command)[0]
    for _, allowed := range allowedCommands {
        if cmd == allowed {
            return nil
        }
    }
    
    return fmt.Errorf("command %q not allowed for security reasons", cmd)
}

// 2. Timeout enforcement
func (d *ServiceDetector) IsRunningWithTimeout(ctx context.Context, serviceName string, timeout time.Duration) (bool, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    return d.IsRunning(ctx, serviceName)
}

// 3. Privilege separation
// - Read-only service status checking (no service management by default)
// - Optional service management with explicit enable flags
// - Minimal required permissions
```

### **Error Handling and Resilience**
```go
type ServiceDetectionError struct {
    ServiceName string
    Platform    string
    Cause       error
    Suggestion  string
}

func (e *ServiceDetectionError) Error() string {
    return fmt.Sprintf("failed to detect service %q on %s: %v. Suggestion: %s", 
        e.ServiceName, e.Platform, e.Cause, e.Suggestion)
}

// Graceful degradation
func (d *ServiceDetector) IsRunningWithFallback(ctx context.Context, serviceName string) (*ServiceInfo, error) {
    // 1. Try native service manager
    if info, err := d.checkNativeManager(ctx, serviceName); err == nil {
        return info, nil
    }
    
    // 2. Fall back to process checking
    if info, err := d.checkProcessName(ctx, serviceName); err == nil {
        return info, nil
    }
    
    // 3. Fall back to package manager
    if info, err := d.checkPackageManager(ctx, serviceName); err == nil {
        return info, nil
    }
    
    // 4. Return unknown status (not an error)
    return &ServiceInfo{
        Name:    serviceName,
        Running: false,
        Healthy: false,
        Version: "unknown",
    }, nil
}
```

---

## 📦 **Provider Integration Design**

### **Enhanced Package Resource**
```go
// Extend existing pkg_package resource with service status
type PackageResourceModel struct {
    // ... existing fields ...
    
    // New computed service status
    ProvidedServices types.List   `tfsdk:"provided_services"`
    ServiceStatus    types.Map    `tfsdk:"service_status"`
    AllServicesUp    types.Bool   `tfsdk:"all_services_up"`
}

// Enhanced package resource output
resource "pkg_package" "colima" {
  name    = "colima"
  state   = "present"
  manager = "brew"
}

output "colima_info" {
  value = {
    installed         = pkg_package.colima.state == "present"
    provided_services = pkg_package.colima.provided_services
    service_status    = pkg_package.colima.service_status
    all_services_up   = pkg_package.colima.all_services_up
  }
}
```

### **Cross-Provider Integration**
```hcl
# Seamless integration with dotfiles provider
module "development_environment" {
  source = "./modules/dev-env"
  
  # Package installation
  packages = {
    colima = { category = "containers" }
    docker = { category = "containers" }
  }
}

# Service status checking
data "pkg_services_overview" "infrastructure" {
  services = keys(module.development_environment.packages)
}

# Dotfiles configuration based on service status
module "dotfiles" {
  source = "./modules/dotfiles"
  
  # Pass service status to dotfiles module
  service_status = data.pkg_services_overview.infrastructure.services
  
  # Dotfiles module can conditionally configure based on services
  configure_docker = data.pkg_services_overview.infrastructure.services.docker.running
  configure_postgres = data.pkg_services_overview.infrastructure.services.postgres.running
}
```

---

## 🎯 **Success Metrics**

### **Technical Success Criteria**
- [ ] **100% Cross-platform compatibility** (macOS, Linux, Windows)
- [ ] **<500ms average response time** for service status checks
- [ ] **99%+ accuracy** in service detection
- [ ] **Zero breaking changes** to existing pkg_package functionality
- [ ] **Comprehensive test coverage** (80%+ unit, integration, acceptance)

### **User Experience Success Criteria**
- [ ] **Idempotent operations** - no unnecessary re-checks
- [ ] **Clear error messages** with actionable suggestions
- [ ] **Structured output** usable in terraform logic
- [ ] **Comprehensive documentation** with real-world examples
- [ ] **Migration guide** from null_resource approaches

### **Integration Success Criteria**
- [ ] **Seamless dotfiles provider integration**
- [ ] **terraform-devenv module compatibility**
- [ ] **Conditional resource creation** based on service status
- [ ] **Service dependency validation**

---

## 🚀 **Implementation Timeline**

### **Sprint 1 (Week 1): Core Foundation**
- [ ] Service detector interface and platform implementations
- [ ] Package-service mapping database
- [ ] Basic `pkg_service_status` data source
- [ ] Unit tests for core functionality

### **Sprint 2 (Week 2): Data Sources**
- [ ] Complete `pkg_service_status` implementation
- [ ] `pkg_services_overview` bulk data source
- [ ] Health checking framework
- [ ] Integration tests

### **Sprint 3 (Week 3): Integration and Polish**
- [ ] Cross-provider integration with dotfiles provider
- [ ] Performance optimization and caching
- [ ] Comprehensive documentation and examples
- [ ] Migration guide and tooling

### **Sprint 4 (Week 4): Advanced Features**
- [ ] Optional service management resources
- [ ] Advanced health checking (HTTP, TCP, custom)
- [ ] Monitoring and metrics integration
- [ ] Production readiness validation

---

## 🎉 **Expected Outcomes**

### **Immediate Benefits**
- ✅ **Terraform-native service checking** replaces shell scripts
- ✅ **Idempotent operations** eliminate unnecessary re-runs
- ✅ **Structured service data** enables conditional logic
- ✅ **Cross-platform consistency** with unified API

### **Long-term Value**
- ✅ **Complete infrastructure management** - packages + services + configurations
- ✅ **Dependency validation** - ensure services are ready before configuration
- ✅ **Environment reliability** - detect and handle service failures gracefully
- ✅ **Developer productivity** - automated service stack management

### **Ecosystem Impact**
- ✅ **Enhanced terraform-devenv** with reliable service management
- ✅ **Better dotfiles integration** with service-aware configurations
- ✅ **Community value** - reusable service management patterns
- ✅ **Infrastructure as Code maturity** - complete environment definition

---

## 📋 **Next Steps**

### **Immediate Actions**
1. **Validate approach** with stakeholders and community feedback
2. **Create GitHub issue** for tracking and community input
3. **Prototype core service detector** for macOS (primary platform)
4. **Design API contracts** with terraform-devenv integration in mind

### **Development Process**
1. **TDD approach** - write tests first for service detection
2. **Platform-by-platform** - start with macOS, then Linux, then Windows
3. **Incremental releases** - ship data sources first, resources later
4. **Community feedback** - gather input from terraform-devenv users

---

**This feature specification provides a comprehensive roadmap for transforming your package provider into a complete package + service management solution, perfectly complementing your symlink-based dotfiles approach.** 🚀

The terraform-native service checking will eliminate the current `null_resource` workarounds while providing much better integration with your Infrastructure as Code development environment management system.

