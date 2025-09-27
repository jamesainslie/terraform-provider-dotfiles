# Fixing Strategy Field Issues in External Modules

## Problem Description

External Terraform modules that use `dotfiles_file` resource with a `strategy` field are experiencing issues because the file resource doesn't process strategy fields - it's designed for direct file operations only.

## Root Cause

The issue occurs when modules like `terraform-devenv` try to use `dotfiles_file` for all strategies:

```hcl
# PROBLEMATIC - This ignores strategy field
resource "dotfiles_file" "application_configs" {
  for_each = merge([
    for app_name, app_config in var.application_configs : {
      for config_name, config_details in app_config.config_mappings :
      "${app_name}_${config_name}" => {
        source_path = config_name
        target_path = config_details.target_path
        strategy   = config_details.strategy  # ← IGNORED
      }
    }
  ]...)
}
```

## Solution: Use Correct Resources

### Option 1: Use dotfiles_application (Recommended)

Replace the problematic pattern with the proper application resource:

```hcl
resource "dotfiles_application" "application_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  # Strategy is properly handled internally
}
```

### Option 2: Dynamic Resource Selection

If you need fine-grained control, use conditional resources:

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
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  create_parents = true
}

# File resources (copy/template)
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
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  is_template = each.value.strategy == "template"
}
```

### Option 3: Add Validation (Temporary Fix)

Until the proper fix is implemented, add validation to prevent unsupported strategies:

```hcl
variable "application_configs" {
  description = "Application configuration file mappings"
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
        contains(["copy"], mapping_config.strategy)
      ]
    ]))
    error_message = "Currently only 'copy' strategy is supported in this module. Use dotfiles_application resource for full strategy support."
  }
}
```

## Migration Steps

1. **Backup existing state**: `terraform state pull > backup.tfstate`
2. **Update module configuration** to use `dotfiles_application`
3. **Import existing resources** if needed
4. **Test the configuration** in a safe environment
5. **Apply changes** to production

## Prevention

Always use the correct resource for your use case:
- `dotfiles_application` - For application configs with strategy support
- `dotfiles_file` - For direct file operations only
- `dotfiles_symlink` - For direct symlink operations only
- `dotfiles_directory` - For directory operations only
