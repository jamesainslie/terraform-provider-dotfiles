# GitHub Repository Example

This example demonstrates how to use the Terraform Dotfiles Provider to manage dotfiles from GitHub repositories with secure authentication.

## Features Demonstrated

-  **Public GitHub repositories** (no authentication required)
-  **Private GitHub repositories** with Personal Access Token authentication
-  **SSH authentication** with private keys
-  **Enterprise GitHub** support
-  **Automatic repository updates** with configurable intervals
-  **Secure credential handling** via environment variables and Terraform variables

## Quick Start

### 1. Set up GitHub Personal Access Token

Create a Personal Access Token (PAT) for GitHub authentication:

1. Go to [GitHub Settings > Personal Access Tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Select appropriate scopes:
   - `repo` - For private repository access
   - `public_repo` - For public repository access only
4. Copy the token (starts with `ghp_`)

### 2. Configure Authentication

Choose one of these authentication methods:

#### Option A: Environment Variables (Recommended)

```bash
export GITHUB_TOKEN="ghp_your_token_here"
export TF_VAR_github_token="$GITHUB_TOKEN"
```

#### Option B: Terraform Variables

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
```

#### Option C: SSH Authentication

```hcl
resource "dotfiles_repository" "ssh_dotfiles" {
  source_path              = "git@github.com:username/dotfiles.git"
  git_ssh_private_key_path = "~/.ssh/id_ed25519"
  git_ssh_passphrase       = var.ssh_passphrase  # Optional
}
```

### 3. Initialize and Apply

```bash
terraform init
terraform plan
terraform apply
```

## Configuration Examples

### Public Repository (No Authentication)

```hcl
resource "dotfiles_repository" "public" {
  name        = "public-dotfiles"
  source_path = "https://github.com/example/dotfiles.git"
  git_branch  = "main"
}
```

### Private Repository with PAT

```hcl
resource "dotfiles_repository" "private" {
  name        = "private-dotfiles"
  source_path = "https://github.com/username/dotfiles.git"
  
  git_personal_access_token = var.github_token
  git_username              = "username"  # Optional
  git_branch                = "main"
}
```

### SSH Authentication

```hcl
resource "dotfiles_repository" "ssh" {
  name        = "ssh-dotfiles"
  source_path = "git@github.com:username/dotfiles.git"
  
  git_ssh_private_key_path = "~/.ssh/id_ed25519"
  git_ssh_passphrase       = var.ssh_passphrase  # If key is encrypted
}
```

### Enterprise GitHub

```hcl
resource "dotfiles_repository" "enterprise" {
  name        = "enterprise-dotfiles"
  source_path = "https://github.enterprise.com/company/dotfiles.git"
  
  git_personal_access_token = var.enterprise_github_token
  git_update_interval      = "30m"  # Update every 30 minutes
}
```

## Security Best Practices

### 1. Never Hardcode Tokens

 **Don't do this:**
```hcl
git_personal_access_token = "ghp_hardcoded_token_here"  # NEVER DO THIS
```

 **Do this instead:**
```hcl
git_personal_access_token = var.github_token            # Use variables
# or let it read from GITHUB_TOKEN environment variable automatically
```

### 2. Use Environment Variables

The provider automatically checks these environment variables:
- `GITHUB_TOKEN`
- `GH_TOKEN`

```bash
export GITHUB_TOKEN="ghp_your_token_here"
terraform apply  # Token used automatically
```

### 3. Secure SSH Keys

Ensure proper permissions on SSH keys:
```bash
chmod 600 ~/.ssh/id_ed25519      # Private key
chmod 644 ~/.ssh/id_ed25519.pub  # Public key
```

### 4. Token Permissions

Use minimal required scopes:
- **Public repos only**: `public_repo`
- **Private repos**: `repo`
- **Organizations**: Add organization access if needed

## Advanced Features

### Automatic Updates

Configure automatic repository updates:

```hcl
resource "dotfiles_repository" "auto_update" {
  name                = "auto-dotfiles"
  source_path         = "https://github.com/username/dotfiles.git"
  git_update_interval = "1h"  # Check every hour
  
  git_personal_access_token = var.github_token
}
```

Update intervals:
- `30s` - Every 30 seconds
- `5m` - Every 5 minutes
- `1h` - Every hour (recommended)
- `12h` - Twice daily
- `24h` - Daily
- `never` - Disable automatic updates

### Branch Management

Target specific branches:

```hcl
resource "dotfiles_repository" "feature_branch" {
  name        = "feature-dotfiles"
  source_path = "https://github.com/username/dotfiles.git"
  git_branch  = "experimental"  # Use specific branch
}
```

### Multiple Repositories

Manage multiple dotfiles repositories:

```hcl
resource "dotfiles_repository" "personal" {
  name        = "personal-dotfiles"
  source_path = "https://github.com/username/personal-dotfiles.git"
}

resource "dotfiles_repository" "work" {
  name        = "work-dotfiles"
  source_path = "https://github.com/company/work-dotfiles.git"
}
```

## Repository Information

Access repository metadata through computed attributes:

```hcl
output "repo_info" {
  value = {
    local_path  = dotfiles_repository.private.local_path
    last_commit = dotfiles_repository.private.last_commit
    last_update = dotfiles_repository.private.last_update
  }
}
```

## Local Cache

Repositories are cached locally at:
- **macOS/Linux**: `~/.terraform-dotfiles-cache/`
- **Windows**: `%USERPROFILE%\.terraform-dotfiles-cache\`

Cache structure:
```
~/.terraform-dotfiles-cache/
└── github.com/
    └── username/
        └── dotfiles/  # Cloned repository
```

## Troubleshooting

### Authentication Issues

1. **"authentication required" error**:
   - Verify your token is valid: `curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user`
   - Check token scopes in GitHub settings

2. **SSH key issues**:
   - Verify SSH key is loaded: `ssh-add -l`
   - Test SSH connection: `ssh -T git@github.com`

3. **Permission errors**:
   - Check SSH key permissions: `ls -la ~/.ssh/`
   - Ensure private key is 600: `chmod 600 ~/.ssh/id_ed25519`

### Repository Issues

1. **"repository not found"**:
   - Verify repository URL is correct
   - Check you have access to the repository
   - For private repos, ensure proper authentication

2. **"branch not found"**:
   - Verify branch name exists in repository
   - Use `main` or `master` depending on repository default

3. **Update failures**:
   - Check network connectivity
   - Verify authentication is still valid
   - Review Terraform logs: `TF_LOG=DEBUG terraform apply`

## Environment Variables

The provider recognizes these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | `ghp_abc123...` |
| `GH_TOKEN` | Alternative GitHub token variable | `ghp_abc123...` |
| `TF_VAR_github_token` | Terraform variable for GitHub token | `ghp_abc123...` |

## Example Output

After successful application:

```bash
$ terraform apply

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.

Outputs:

repository_info = {
  "private_repo" = {
    "id" = "private-dotfiles"
    "last_commit" = "a1b2c3d4e5f6789abcdef1234567890abcdef123"
    "last_update" = "2024-01-15T10:30:00Z"
    "local_path" = "/Users/username/.terraform-dotfiles-cache/github.com/username/dotfiles"
  }
}

system_info = {
  "architecture" = "arm64"
  "config_dir" = "/Users/username/.config"
  "home_dir" = "/Users/username"
  "platform" = "macos"
}
```

## Next Steps

After setting up your repositories, you can:

1. **Create file resources** to manage specific files
2. **Set up symlinks** to link configurations to their target locations  
3. **Use templates** for dynamic configuration generation
4. **Configure backups** to protect existing files

See the other examples in this repository for more advanced usage patterns.
