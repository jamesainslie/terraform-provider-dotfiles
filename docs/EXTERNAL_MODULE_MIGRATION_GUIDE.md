# External Module Migration Guide: Strategy Field Removal

**Target Audience**: External Terraform module maintainers using terraform-provider-dotfiles  
**Impact Level**: Breaking Change for modules using deprecated `strategy` field  
**Migration Timeline**: Immediate action recommended  
**Provider Version**: v1.1.0+ (strategy field removed)

## Executive Summary

The terraform-provider-dotfiles has implemented a **breaking change** by removing the non-functional `strategy` field from the `dotfiles_file` resource. This change eliminates a critical bug where the strategy field was captured but ignored, leading to runtime failures and user confusion.

**Key Changes:**
- ❌ `dotfiles_file` no longer accepts `strategy` field
- ✅ Clear, explicit resource architecture implemented  
- 🛠️ Comprehensive migration tools provided
- 📚 Updated documentation and examples

## Problem Statement

### Original Issue
External modules (like `terraform-devenv`) were implementing patterns like this:

```hcl
# BROKEN PATTERN - strategy field was ignored
resource "dotfiles_file" "application_configs" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
        strategy   = config_details.strategy  # ← IGNORED!
      }
    }
  ]...)
  
  # Always performed copy operations regardless of strategy
}
```

### Impact
- **Runtime Failures**: `strategy = "symlink"` with directories caused copy failures
- **User Confusion**: Configuration appeared correct but didn't work as expected  
- **Inconsistent Behavior**: No feedback that strategy was being ignored

## Solution: Explicit Resource Architecture

The provider now enforces explicit resource selection:

| Resource | Purpose | When to Use |
|----------|---------|-------------|
| `dotfiles_file` | **Copy operations only** | Single files, templating, direct file operations |
| `dotfiles_symlink` | **Symlink operations only** | Files or directories that should be symlinked |
| `dotfiles_directory` | **Directory operations only** | Recursive directory synchronization |
| `dotfiles_application` | **Multi-strategy configs** | Complex application configurations |

## Migration Strategies

### Strategy 1: Use dotfiles_application (Recommended)

Replace strategy-based `dotfiles_file` resources with the proper `dotfiles_application` resource:

**Before (Broken):**
```hcl
resource "dotfiles_file" "application_configs" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
        strategy   = config_details.strategy
      }
    }
  ]...)
  
  # Configuration continues...
}
```

**After (Correct):**
```hcl
resource "dotfiles_application" "application_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  # Strategy is handled internally by the resource
  # No complex for_each logic needed
  # Built-in template path support
  # Proper error handling for all strategies
}
```

**Benefits:**
- ✅ Strategy field properly implemented and functional
- ✅ Simplified module logic (no complex `for_each` merging)
- ✅ Built-in template path expansion (`{{.config_dir}}`, `{{.application}}`, etc.)
- ✅ Better error messages and validation
- ✅ Handles edge cases (directories, permissions, etc.)

### Strategy 2: Dynamic Resource Selection

If you need granular control, separate resources by strategy:

```hcl
# Symlink resources
resource "dotfiles_symlink" "application_symlinks" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
      } if config_details.strategy == "symlink"
    }
  ]...)
  
  repository     = local.repository_id
  name           = "${each.key}-symlink"
  source_path    = each.value.source_path
  target_path    = each.value.target_path
  create_parents = true
}

# File resources (copy and template)
resource "dotfiles_file" "application_files" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
        strategy   = config_details.strategy
      } if contains(["copy", "template"], config_details.strategy)
    }
  ]...)
  
  repository  = local.repository_id
  name        = "${each.key}-file"
  source_path = each.value.source_path
  target_path = each.value.target_path
  is_template = each.value.strategy == "template"
}
```

**Benefits:**
- ✅ Maximum control over resource configuration
- ✅ Can handle complex edge cases
- ✅ Allows per-resource customization

**Drawbacks:**
- ❌ More complex configuration
- ❌ Requires maintaining multiple resource blocks
- ❌ Manual handling of strategy logic

### Strategy 3: Validation During Transition

Add validation to prevent users from using unsupported strategies during migration:

```hcl
variable "application_configs" {
  description = "Application configuration mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
  default = {}
  
  validation {
    condition = alltrue(flatten([
      for app_name, app_config in var.application_configs : [
        for mapping_name, mapping_config in app_config.config_mappings :
        contains(["copy", "symlink", "template"], mapping_config.strategy)
      ]
    ]))
    error_message = "Strategy must be one of: copy, symlink, template. Note: This module is being migrated to use dotfiles_application for proper strategy support."
  }
}
```

## Step-by-Step Migration Process

### Phase 1: Assessment (Immediate)
1. **Identify Impact**: Search your modules for `dotfiles_file` resources with `strategy` fields
2. **Catalog Usage**: Document which strategies are used and how
3. **Test Current Behavior**: Verify what actually works vs. what users expect

### Phase 2: Implementation (1-2 weeks)
1. **Choose Strategy**: Select Strategy 1 (recommended) or Strategy 2 based on your needs
2. **Update Resources**: Replace problematic resources with correct ones
3. **Update Variables**: Ensure variable structure supports new approach
4. **Add Validation**: Implement input validation to catch issues early

### Phase 3: Testing (1 week)
1. **Unit Testing**: Test module with various strategy combinations
2. **Integration Testing**: Test with actual dotfiles repositories
3. **Edge Case Testing**: Test directories, permissions, templates
4. **Backward Compatibility**: Ensure existing users can migrate smoothly

### Phase 4: Documentation & Release (1 week)
1. **Update Examples**: Provide clear before/after examples
2. **Migration Guide**: Create user-facing migration documentation
3. **Release Notes**: Document breaking changes and migration path
4. **Support Plan**: Prepare to help users with migration issues

## Testing Strategy

### Automated Testing
```hcl
# Test all strategy types work correctly
module "test_strategies" {
  source = "../"
  
  application_configs = {
    "testapp" = {
      config_mappings = {
        "copy-file.conf" = {
          target_path = "/tmp/test-copy.conf"
          strategy   = "copy"
        }
        "symlink-dir" = {
          target_path = "/tmp/test-symlink"
          strategy   = "symlink"
        }
        "template.conf.tmpl" = {
          target_path = "/tmp/test-template.conf"
          strategy   = "template"
        }
      }
    }
  }
}

# Verify correct resources are created
check "strategy_handling" {
  # Should create dotfiles_application resource, not dotfiles_file
  assert {
    condition = length([
      for r in values(module.test_strategies) : r
      if startswith(r.type, "dotfiles_application")
    ]) > 0
    error_message = "Module should use dotfiles_application for strategy handling"
  }
}
```

### Manual Testing Checklist
- [ ] Copy strategy creates files correctly
- [ ] Symlink strategy creates symlinks correctly  
- [ ] Template strategy processes templates correctly
- [ ] Directory sources work with symlink strategy
- [ ] Error messages are clear and helpful
- [ ] Performance is acceptable with large configurations

## Migration Tools

### Automated Migration Tool
The provider includes a migration tool to help convert existing configurations:

```bash
# Validate existing configuration
go run github.com/jamesainslie/terraform-provider-dotfiles/cmd/migrate-config validate main.tf

# Migrate configuration  
go run github.com/jamesainslie/terraform-provider-dotfiles/cmd/migrate-config migrate main.tf main-migrated.tf
```

### Manual Validation Script
```bash
#!/bin/bash
# Check for problematic patterns in Terraform files

echo "Checking for dotfiles_file resources with strategy field..."
grep -r "dotfiles_file.*{" . --include="*.tf" | while read -r file; do
  if grep -A 10 "$file" | grep -q "strategy.*="; then
    echo "⚠️  Found strategy field usage in: $file"
  fi
done

echo "✅ Validation complete"
```

## Variable Structure Compatibility

The variable structure for `dotfiles_application` is compatible with existing patterns:

```hcl
# This structure works with both old and new approaches
variable "application_configs" {
  description = "Application configuration mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
  default = {}
}

# Example usage
application_configs = {
  "vscode" = {
    config_mappings = {
      "vscode/settings.json" = {
        target_path = "~/Library/Application Support/Code/User/settings.json"
        strategy   = "copy"
      }
      "vscode/extensions/" = {
        target_path = "~/Library/Application Support/Code/User/extensions/"  
        strategy   = "symlink"
      }
    }
  }
}
```

## Common Migration Issues & Solutions

### Issue 1: Complex for_each Logic
**Problem**: Existing complex `for_each` with strategy filtering
**Solution**: Use `dotfiles_application` which handles this internally

### Issue 2: Custom Resource Names
**Problem**: Need specific resource naming patterns  
**Solution**: Use `name` attribute in `dotfiles_application` or implement Strategy 2

### Issue 3: Per-File Configuration
**Problem**: Need different settings per file (permissions, etc.)
**Solution**: Use Strategy 2 with separate resource blocks for fine control

### Issue 4: Template Path Variables
**Problem**: Using custom template path logic
**Solution**: `dotfiles_application` has built-in template variables (`{{.config_dir}}`, `{{.application}}`)

### Issue 5: Backwards Compatibility
**Problem**: Existing users have working configurations
**Solution**: Phase migration with validation, provide clear upgrade path

## Timeline & Support

### Recommended Timeline
- **Week 1-2**: Assessment and planning
- **Week 3-4**: Implementation and testing  
- **Week 5**: Documentation and release preparation
- **Week 6**: Release and user support

### Provider Support
- **Migration Tools**: Automated validation and conversion tools provided
- **Documentation**: Comprehensive guides and examples available
- **Community Support**: GitHub issues for migration assistance
- **Backwards Compatibility**: Old configurations will show clear error messages

### Breaking Change Policy
- **Provider Version**: Strategy field removed in v1.1.0+
- **Deprecation Period**: Field was non-functional, so immediate removal acceptable
- **Migration Support**: Tools and documentation provided for smooth transition

## Example: Complete Module Migration

### Before (terraform-devenv/modules/dotfiles/main.tf)
```hcl
# Note: This was a placeholder - strategy field was ignored
resource "dotfiles_file" "application_configs" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
        strategy   = config_details.strategy  # IGNORED
      }
    }
  ]...)
  
  repository  = dotfiles_repository.main.id
  name        = each.key
  source_path = each.value.source_path
  target_path = each.value.target_path
  # Strategy was captured but never used
}
```

### After (Recommended Approach)
```hcl
resource "dotfiles_application" "application_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  # Strategy is now properly implemented:
  # - "copy": copies files/directories
  # - "symlink": creates symlinks to files/directories  
  # - "template": processes templates then copies
  
  # Depends on repository setup
  depends_on = [dotfiles_repository.main]
}
```

### Variable Updates (No Changes Required)
```hcl
# This variable structure works with both approaches
variable "application_configs" {
  description = "Application configuration file mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
  default = {}
}
```

## Validation & Quality Assurance

### Pre-Migration Checklist
- [ ] Identify all `dotfiles_file` resources using `strategy` field
- [ ] Document current user expectations vs. actual behavior
- [ ] Plan testing strategy for all supported strategies
- [ ] Prepare user communication and migration timeline

### Post-Migration Checklist  
- [ ] All strategy combinations work correctly
- [ ] Error messages are helpful and actionable
- [ ] Performance is acceptable
- [ ] Documentation is complete and accurate
- [ ] Migration path is clear for existing users
- [ ] Support resources are prepared

## Conclusion

The strategy field removal eliminates a critical source of confusion and bugs in external modules. While this is a breaking change, the new explicit resource architecture provides:

- **Clarity**: Each resource has a single, clear purpose
- **Reliability**: No more ignored configuration fields
- **Power**: `dotfiles_application` properly implements all strategies
- **Maintainability**: Simpler, more predictable code

The migration is straightforward with provided tools and documentation. We recommend **Strategy 1** (using `dotfiles_application`) for most use cases as it provides the best balance of functionality and simplicity.

## Support Resources

- **Migration Tool**: `cmd/migrate-config/main.go` in provider repository
- **Documentation**: [EXTERNAL_MODULE_FIX.md](EXTERNAL_MODULE_FIX.md)  
- **Examples**: `examples/migration/` directory
- **Community**: GitHub Issues for migration assistance

**Questions?** Open an issue in the terraform-provider-dotfiles repository with the `migration-support` label.
