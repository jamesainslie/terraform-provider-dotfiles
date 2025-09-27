# Module Author Guide: Avoiding Common Resource Misuse

This guide helps external Terraform module authors use the terraform-provider-dotfiles correctly and avoid common pitfalls that can lead to confusing errors.

## Quick Reference: Resource Selection

| Scenario | ❌ Wrong Approach | ✅ Correct Approach |
|----------|------------------|---------------------|
| **Application configs with strategy** | `dotfiles_file` with `strategy` field | `dotfiles_application` resource |
| **Dynamic resource based on strategy** | Complex `for_each` with `dotfiles_file` | `dotfiles_application` or separate resources |
| **Mix of symlinks and copies** | Single resource type with strategy logic | `dotfiles_application` or resource per strategy |

## Common Anti-Pattern: The Strategy Field Trap

### ❌ Problem: Using dotfiles_file with strategy

Many modules attempt to use `dotfiles_file` with a `strategy` field, expecting it to choose between copy/symlink behavior:

```hcl
# THIS DOESN'T WORK - strategy field is ignored
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
  
  # This ALWAYS does copy operations, regardless of strategy field
  source_path = each.value.source_path
  target_path = each.value.target_path
  # strategy field exists but is ignored
}
```

**Why this fails:**
- `dotfiles_file` is designed for copy operations only
- The `strategy` field is deprecated and ignored
- Users expect symlink behavior but get copy behavior
- Directories will fail with copy operations

### ✅ Solution 1: Use dotfiles_application (Recommended)

The `dotfiles_application` resource is specifically designed for strategy-aware deployment:

```hcl
resource "dotfiles_application" "application_configs" {
  for_each = var.application_configs
  
  application_name = each.key
  config_mappings = each.value.config_mappings
  
  # Strategy is handled internally based on config_mappings
  # No need for complex logic - the resource handles it
}
```

**Variable structure:**
```hcl
variable "application_configs" {
  description = "Application configuration mappings"
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string  # "symlink", "copy", or "template"
    }))
  }))
}
```

### ✅ Solution 2: Dynamic Resource Selection

If you need more granular control, separate resources by strategy:

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
  
  source_path = each.value.source_path
  target_path = each.value.target_path
  is_template = each.value.strategy == "template"
}
```

## Validation and Error Prevention

### Add Input Validation

Prevent users from providing unsupported configurations:

```hcl
variable "application_configs" {
  # ... type definition ...
  
  validation {
    condition = alltrue(flatten([
      for app_name, app_config in var.application_configs : [
        for mapping_name, mapping_config in app_config.config_mappings :
        contains(["symlink", "copy", "template"], mapping_config.strategy)
      ]
    ]))
    error_message = "Strategy must be one of: symlink, copy, template"
  }
}
```

### Provide Clear Error Messages

When using the incorrect pattern, users will see provider warnings:

```
│ Warning: Strategy field ignored in dotfiles_file resource
│ 
│   with dotfiles_file.application_configs["app_config"],
│   on main.tf line 10, in resource "dotfiles_file" "application_configs":
│   10: resource "dotfiles_file" "application_configs" {
│ 
│ The 'strategy = "symlink"' field is not supported by dotfiles_file resource and will be ignored.
│ Use the `dotfiles_symlink` resource or `dotfiles_application` resource with strategy = "symlink" for 
│ symlink-based deployment. For comprehensive application configuration management, consider using the 
│ `dotfiles_application` resource which properly handles different deployment strategies.
```

## Testing Your Module

### Test All Strategies

Ensure your module handles all expected strategies correctly:

```hcl
# Test configuration
application_configs = {
  "testapp" = {
    config_mappings = {
      "single-file.conf" = {
        target_path = "~/.config/testapp/single.conf"
        strategy   = "copy"
      }
      "symlink-dir/" = {
        target_path = "~/.config/testapp/symlink-dir/"
        strategy   = "symlink"
      }
      "template.conf.tmpl" = {
        target_path = "~/.config/testapp/rendered.conf"
        strategy   = "template"
      }
    }
  }
}
```

### Verify Resource Creation

After applying, verify the correct resources are created:

```bash
# Should show dotfiles_application resources, NOT dotfiles_file with strategy
terraform state list | grep dotfiles_
```

## Migration from Problematic Patterns

If your module currently uses the problematic pattern:

### Step 1: Update Resource Definition

```hcl
# OLD - Remove this
resource "dotfiles_file" "application_configs" {
  # ... with strategy field
}

# NEW - Add this
resource "dotfiles_application" "application_configs" {
  for_each = var.application_configs
  application_name = each.key
  config_mappings = each.value.config_mappings
}
```

### Step 2: Update Variables (if needed)

The variable structure remains the same for `dotfiles_application`:

```hcl
# This structure works for dotfiles_application
variable "application_configs" {
  type = map(object({
    config_mappings = map(object({
      target_path = string
      strategy   = string
    }))
  }))
}
```

### Step 3: Test Migration

1. Create a test environment
2. Apply the old configuration
3. Update to new configuration
4. Plan and verify changes
5. Apply and verify functionality

## Additional Resources

- [dotfiles_application documentation](resources/application.md)
- [Resource selection guide](README.md#resource-selection-guide)
- [External module fix guide](EXTERNAL_MODULE_FIX.md)

## Need Help?

If you're migrating an existing module and encounter issues:

1. Check the provider warnings in `terraform plan` output
2. Review the [External Module Fix Guide](EXTERNAL_MODULE_FIX.md)
3. Test with a simple configuration first
4. Consider using `dotfiles_application` for most use cases

The provider is designed to guide you toward correct usage through helpful warnings and clear documentation.
