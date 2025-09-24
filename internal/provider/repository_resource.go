// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/git"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RepositoryResource{}

func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	client *DotfilesClient
}

// RepositoryResourceModel describes the resource data model.
type RepositoryResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	SourcePath           types.String `tfsdk:"source_path"`
	Description          types.String `tfsdk:"description"`
	DefaultBackupEnabled types.Bool   `tfsdk:"default_backup_enabled"`
	DefaultFileMode      types.String `tfsdk:"default_file_mode"`
	DefaultDirMode       types.String `tfsdk:"default_dir_mode"`

	// Git-specific attributes
	GitBranch              types.String `tfsdk:"git_branch"`
	GitPersonalAccessToken types.String `tfsdk:"git_personal_access_token"`
	GitUsername            types.String `tfsdk:"git_username"`
	GitSSHPrivateKeyPath   types.String `tfsdk:"git_ssh_private_key_path"`
	GitSSHPassphrase       types.String `tfsdk:"git_ssh_passphrase"`
	GitUpdateInterval      types.String `tfsdk:"git_update_interval"`

	// Computed attributes
	LocalPath  types.String `tfsdk:"local_path"`
	LastCommit types.String `tfsdk:"last_commit"`
	LastUpdate types.String `tfsdk:"last_update"`
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a dotfiles repository configuration. Supports both local paths and Git repositories (GitHub, GitLab, etc.)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository identifier",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository name",
			},
			"source_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the dotfiles repository. Can be a local path or Git URL (e.g., 'https://github.com/user/dotfiles.git')",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Repository description",
			},
			"default_backup_enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Default backup setting for resources in this repository",
			},
			"default_file_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Default file permissions (e.g., '0644')",
			},
			"default_dir_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Default directory permissions (e.g., '0755')",
			},

			// Git-specific attributes
			"git_branch": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Git branch to checkout (defaults to repository default branch)",
			},
			"git_personal_access_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "GitHub Personal Access Token for private repository authentication",
			},
			"git_username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for Git authentication (optional when using PAT)",
			},
			"git_ssh_private_key_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to SSH private key for Git authentication",
			},
			"git_ssh_passphrase": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Passphrase for SSH private key",
			},
			"git_update_interval": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Interval to check for updates (e.g., '1h', '30m'). Use 'never' to disable automatic updates",
			},

			// Computed attributes
			"local_path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Local path where the repository is stored",
			},
			"last_commit": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SHA of the last commit",
			},
			"last_update": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp of the last repository update",
			},
		},
	}
}

func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*DotfilesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *DotfilesClient, got something else. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating repository resource", map[string]interface{}{
		"name":        data.Name.ValueString(),
		"source_path": data.SourcePath.ValueString(),
	})

	sourcePath := data.SourcePath.ValueString()

	// Check if source is a Git URL
	if git.IsGitURL(sourcePath) {
		// Handle Git repository
		info, err := r.setupGitRepository(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to setup Git repository",
				fmt.Sprintf("Could not clone or setup Git repository: %s", err.Error()),
			)
			return
		}

		// Update model with Git info
		data.LocalPath = types.StringValue(info.LocalPath)
		data.LastCommit = types.StringValue(info.LastCommit)
		data.LastUpdate = types.StringValue(info.LastUpdate.Format(time.RFC3339))

		tflog.Info(ctx, "Git repository cloned successfully", map[string]interface{}{
			"url":         info.URL,
			"local_path":  info.LocalPath,
			"last_commit": info.LastCommit,
			"last_update": data.LastUpdate.ValueString(),
		})
	} else {
		// Handle local repository
		err := r.setupLocalRepository(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to setup local repository",
				fmt.Sprintf("Could not setup local repository: %s", err.Error()),
			)
			return
		}

		data.LocalPath = data.SourcePath
		data.LastUpdate = types.StringValue(time.Now().Format(time.RFC3339))

		// Check if local repository is a Git repository and get commit info
		localPath := data.SourcePath.ValueString()

		tflog.Debug(ctx, "Checking if repository is a Git repository", map[string]interface{}{
			"local_path": localPath,
		})

		isGitRepo := r.isGitRepository(localPath)
		tflog.Debug(ctx, "Git repository detection result", map[string]interface{}{
			"local_path": localPath,
			"is_git":     isGitRepo,
		})

		if isGitRepo {
			tflog.Debug(ctx, "Creating Git manager for local repository", map[string]interface{}{
				"local_path": localPath,
			})

			gitManager, err := git.NewGitManager(nil) // No auth needed for local repos
			if err == nil {
				tflog.Debug(ctx, "Git manager created successfully, retrieving repository info", map[string]interface{}{
					"local_path": localPath,
				})

				info, err := gitManager.GetRepositoryInfo(localPath)
				if err == nil {
					data.LastCommit = types.StringValue(info.LastCommit)
					tflog.Info(ctx, "Successfully retrieved Git info for local repository", map[string]interface{}{
						"local_path":  localPath,
						"last_commit": info.LastCommit,
					})
				} else {
					tflog.Error(ctx, "Failed to get Git repository info", map[string]interface{}{
						"error":      err.Error(),
						"local_path": localPath,
					})
					// Set empty but valid commit for repositories with Git issues
					data.LastCommit = types.StringValue("")
				}
			} else {
				tflog.Error(ctx, "Failed to create Git manager", map[string]interface{}{
					"error":      err.Error(),
					"local_path": localPath,
				})
				data.LastCommit = types.StringValue("")
			}
		} else {
			tflog.Info(ctx, "Repository is not a Git repository, setting empty commit", map[string]interface{}{
				"local_path": localPath,
			})
			// Not a Git repository, set empty but valid commit
			data.LastCommit = types.StringValue("")
		}

		tflog.Info(ctx, "Local repository setup successfully", map[string]interface{}{
			"source_path": sourcePath,
			"local_path":  data.LocalPath.ValueString(),
			"last_commit": data.LastCommit.ValueString(),
			"last_update": data.LastUpdate.ValueString(),
		})
	}

	// Set ID and save state
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading repository resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Check if this is a Git repository
	sourcePath := data.SourcePath.ValueString()
	if git.IsGitURL(sourcePath) {
		// Handle Git repository
		if !data.LocalPath.IsNull() {
			localPath := data.LocalPath.ValueString()

			// Check if local repository still exists
			if _, err := os.Stat(localPath); err == nil {
				// Get current repository info
				authConfig := r.buildAuthConfig(&data)
				gitManager, err := git.NewGitManager(authConfig)
				if err != nil {
					tflog.Warn(ctx, "Failed to create Git manager during read", map[string]interface{}{
						"error": err.Error(),
					})
				} else {
					info, err := gitManager.GetRepositoryInfo(localPath)
					if err == nil {
						// Update computed attributes
						data.LastCommit = types.StringValue(info.LastCommit)
						data.LastUpdate = types.StringValue(info.LastUpdate.Format(time.RFC3339))
					} else {
						tflog.Warn(ctx, "Failed to get repository info", map[string]interface{}{
							"error": err.Error(),
						})
					}
				}
			} else {
				tflog.Warn(ctx, "Local repository path no longer exists", map[string]interface{}{
					"local_path": localPath,
				})
				// Repository was deleted externally, mark for recreation
			}
		}
	} else {
		// Handle local repository - just verify it still exists
		localPath := data.LocalPath.ValueString()
		if localPath == "" {
			localPath = sourcePath
		}

		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			resp.Diagnostics.AddWarning(
				"Local repository not found",
				fmt.Sprintf("The local repository at %s no longer exists.", localPath),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating repository resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	sourcePath := data.SourcePath.ValueString()

	// Check if source is a Git URL and needs updating
	if git.IsGitURL(sourcePath) {
		// Handle Git repository update
		if !data.LocalPath.IsNull() {
			localPath := data.LocalPath.ValueString()

			// Check if local repository exists
			if _, err := os.Stat(localPath); err == nil {
				// Update the repository
				authConfig := r.buildAuthConfig(&data)
				gitManager, err := git.NewGitManager(authConfig)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to create Git manager",
						fmt.Sprintf("Could not create Git manager for update: %s", err.Error()),
					)
					return
				}

				info, err := gitManager.UpdateRepository(ctx, localPath)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to update Git repository",
						fmt.Sprintf("Could not update Git repository: %s", err.Error()),
					)
					return
				}

				// Update computed attributes
				data.LastCommit = types.StringValue(info.LastCommit)
				data.LastUpdate = types.StringValue(info.LastUpdate.Format(time.RFC3339))

				tflog.Info(ctx, "Git repository updated successfully", map[string]interface{}{
					"local_path":  info.LocalPath,
					"last_commit": info.LastCommit,
				})
			} else {
				// Repository doesn't exist locally, re-create it
				tflog.Info(ctx, "Local repository not found, re-cloning", map[string]interface{}{
					"local_path": localPath,
				})

				info, err := r.setupGitRepository(ctx, &data)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to re-setup Git repository",
						fmt.Sprintf("Could not re-clone Git repository: %s", err.Error()),
					)
					return
				}

				// Update model with Git info
				data.LocalPath = types.StringValue(info.LocalPath)
				data.LastCommit = types.StringValue(info.LastCommit)
				data.LastUpdate = types.StringValue(info.LastUpdate.Format(time.RFC3339))
			}
		} else {
			// No local path set, treat as new setup
			info, err := r.setupGitRepository(ctx, &data)
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to setup Git repository",
					fmt.Sprintf("Could not setup Git repository: %s", err.Error()),
				)
				return
			}

			// Update model with Git info
			data.LocalPath = types.StringValue(info.LocalPath)
			data.LastCommit = types.StringValue(info.LastCommit)
			data.LastUpdate = types.StringValue(info.LastUpdate.Format(time.RFC3339))
		}
	} else {
		// Handle local repository update
		err := r.setupLocalRepository(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to update local repository",
				fmt.Sprintf("Could not update local repository: %s", err.Error()),
			)
			return
		}

		data.LocalPath = data.SourcePath
		data.LastUpdate = types.StringValue(time.Now().Format(time.RFC3339))

		// Check if local repository is a Git repository and get commit info
		localPath := data.SourcePath.ValueString()
		if r.isGitRepository(localPath) {
			gitManager, err := git.NewGitManager(nil) // No auth needed for local repos
			if err == nil {
				info, err := gitManager.GetRepositoryInfo(localPath)
				if err == nil {
					data.LastCommit = types.StringValue(info.LastCommit)
					tflog.Debug(ctx, "Retrieved Git info for local repository update", map[string]interface{}{
						"local_path":  localPath,
						"last_commit": info.LastCommit,
					})
				} else {
					tflog.Warn(ctx, "Failed to get Git info for local repository update", map[string]interface{}{
						"error": err.Error(),
					})
					// Set empty but valid commit for non-Git local repos
					data.LastCommit = types.StringValue("")
				}
			} else {
				data.LastCommit = types.StringValue("")
			}
		} else {
			// Not a Git repository, set empty but valid commit
			data.LastCommit = types.StringValue("")
		}
	}

	// Set ID and save state
	data.ID = data.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting repository resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// For dotfiles repositories, we typically don't delete the actual files,
	// just remove them from Terraform state. The local cache will remain.
	tflog.Info(ctx, "Repository resource removed from state", map[string]interface{}{
		"name": data.Name.ValueString(),
	})
}

// setupGitRepository handles cloning and setting up a Git repository.
func (r *RepositoryResource) setupGitRepository(ctx context.Context, data *RepositoryResourceModel) (*git.RepositoryInfo, error) {
	sourcePath := data.SourcePath.ValueString()

	// Create authentication config
	authConfig := &git.AuthConfig{}

	// Configure PAT authentication
	if !data.GitPersonalAccessToken.IsNull() {
		authConfig.PersonalAccessToken = data.GitPersonalAccessToken.ValueString()
		if !data.GitUsername.IsNull() {
			authConfig.Username = data.GitUsername.ValueString()
		}
	}

	// Configure SSH authentication
	if !data.GitSSHPrivateKeyPath.IsNull() {
		authConfig.SSHPrivateKeyPath = data.GitSSHPrivateKeyPath.ValueString()
		if !data.GitSSHPassphrase.IsNull() {
			authConfig.SSHPassphrase = data.GitSSHPassphrase.ValueString()
		}
	}

	// Check for environment variable PAT if not provided
	if authConfig.PersonalAccessToken == "" {
		if envPAT := os.Getenv("GITHUB_TOKEN"); envPAT != "" {
			authConfig.PersonalAccessToken = envPAT
			tflog.Debug(ctx, "Using GitHub token from GITHUB_TOKEN environment variable")
		} else if envPAT := os.Getenv("GH_TOKEN"); envPAT != "" {
			authConfig.PersonalAccessToken = envPAT
			tflog.Debug(ctx, "Using GitHub token from GH_TOKEN environment variable")
		}
	}

	// Create Git manager
	gitManager, err := git.NewGitManager(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git manager: %w", err)
	}

	// Determine local cache path
	cacheRoot := filepath.Join(r.client.HomeDir, ".terraform-dotfiles-cache")
	localPath, err := git.GetLocalCachePath(cacheRoot, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine cache path: %w", err)
	}

	// Check if repository already exists locally
	if _, err := os.Stat(localPath); err == nil {
		// Repository exists, try to update it
		tflog.Debug(ctx, "Repository already exists locally, updating", map[string]interface{}{
			"local_path": localPath,
		})

		info, err := gitManager.UpdateRepository(ctx, localPath)
		if err != nil {
			tflog.Warn(ctx, "Failed to update existing repository, will re-clone", map[string]interface{}{
				"error": err.Error(),
			})

			// Remove existing directory and re-clone
			if err := os.RemoveAll(localPath); err != nil {
				return nil, fmt.Errorf("failed to remove existing repository: %w", err)
			}
		} else {
			return info, nil
		}
	}

	// Clone repository
	branch := ""
	if !data.GitBranch.IsNull() {
		branch = data.GitBranch.ValueString()
	}

	tflog.Debug(ctx, "Cloning Git repository", map[string]interface{}{
		"url":        sourcePath,
		"local_path": localPath,
		"branch":     branch,
	})

	info, err := gitManager.CloneRepository(ctx, sourcePath, localPath, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return info, nil
}

// setupLocalRepository handles validation of a local repository.
func (r *RepositoryResource) setupLocalRepository(ctx context.Context, data *RepositoryResourceModel) error {
	sourcePath := data.SourcePath.ValueString()

	// Expand the path
	if sourcePath[0] == '~' {
		sourcePath = filepath.Join(r.client.HomeDir, sourcePath[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", absPath)
	}

	// Check if it's a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", absPath)
	}

	tflog.Debug(ctx, "Local repository validated", map[string]interface{}{
		"source_path": sourcePath,
		"abs_path":    absPath,
	})

	// Update source path to absolute path
	data.SourcePath = types.StringValue(absPath)

	return nil
}

// buildAuthConfig creates authentication config from resource data.
func (r *RepositoryResource) buildAuthConfig(data *RepositoryResourceModel) *git.AuthConfig {
	authConfig := &git.AuthConfig{}

	// Configure PAT authentication
	if !data.GitPersonalAccessToken.IsNull() {
		authConfig.PersonalAccessToken = data.GitPersonalAccessToken.ValueString()
		if !data.GitUsername.IsNull() {
			authConfig.Username = data.GitUsername.ValueString()
		}
	}

	// Configure SSH authentication
	if !data.GitSSHPrivateKeyPath.IsNull() {
		authConfig.SSHPrivateKeyPath = data.GitSSHPrivateKeyPath.ValueString()
		if !data.GitSSHPassphrase.IsNull() {
			authConfig.SSHPassphrase = data.GitSSHPassphrase.ValueString()
		}
	}

	// Check for environment variable PAT if not provided
	if authConfig.PersonalAccessToken == "" {
		if envPAT := os.Getenv("GITHUB_TOKEN"); envPAT != "" {
			authConfig.PersonalAccessToken = envPAT
		} else if envPAT := os.Getenv("GH_TOKEN"); envPAT != "" {
			authConfig.PersonalAccessToken = envPAT
		}
	}

	return authConfig
}

// isGitRepository checks if a local path contains a Git repository.
func (r *RepositoryResource) isGitRepository(localPath string) bool {
	gitDir := filepath.Join(localPath, ".git")

	stat, err := os.Stat(gitDir)
	if err != nil {
		// .git directory/file doesn't exist
		return false
	}

	// .git exists, check if it's a directory (normal repo) or file (worktree/submodule)
	isGitRepo := stat.IsDir() || stat.Mode().IsRegular()
	return isGitRepo
}
