// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

// Strategy constants define the deployment strategies available.
const (
	StrategySymlink  = "symlink"
	StrategyCopy     = "copy"
	StrategyTemplate = "template"
)

// ValidStrategies contains all valid deployment strategies.
var ValidStrategies = []string{
	StrategySymlink,
	StrategyCopy,
	StrategyTemplate,
}

// ConflictResolution constants define how to handle conflicts.
const (
	ConflictResolutionBackup    = "backup"
	ConflictResolutionOverwrite = "overwrite"
	ConflictResolutionSkip      = "skip"
	ConflictResolutionPrompt    = "prompt"
)

// ValidConflictResolutions contains all valid conflict resolution strategies.
var ValidConflictResolutions = []string{
	ConflictResolutionBackup,
	ConflictResolutionOverwrite,
	ConflictResolutionSkip,
	ConflictResolutionPrompt,
}

// Platform constants define the supported platforms.
const (
	PlatformAuto    = "auto"
	PlatformMacOS   = "macos"
	PlatformLinux   = "linux"
	PlatformWindows = "windows"
)

// ValidPlatforms contains all valid platform identifiers.
var ValidPlatforms = []string{
	PlatformAuto,
	PlatformMacOS,
	PlatformLinux,
	PlatformWindows,
}

// TemplateEngine constants define the supported template engines.
const (
	TemplateEngineGo         = "go"
	TemplateEngineHandlebars = "handlebars"
	TemplateEngineMustache   = "mustache"
	TemplateEngineNone       = "none"
)

// ValidTemplateEngines contains all valid template engines.
var ValidTemplateEngines = []string{
	TemplateEngineGo,
	TemplateEngineHandlebars,
	TemplateEngineMustache,
	TemplateEngineNone,
}

// LogLevel constants define the supported log levels.
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// ValidLogLevels contains all valid log levels.
var ValidLogLevels = []string{
	LogLevelDebug,
	LogLevelInfo,
	LogLevelWarn,
	LogLevelError,
}

// BackupFormat constants define the supported backup formats.
const (
	BackupFormatTimestamped = "timestamped"
	BackupFormatNumbered    = "numbered"
	BackupFormatGitStyle    = "git-style"
)

// ValidBackupFormats contains all valid backup formats.
var ValidBackupFormats = []string{
	BackupFormatTimestamped,
	BackupFormatNumbered,
	BackupFormatGitStyle,
}

// ApplicationDetectionMethod constants define the supported detection methods.
const (
	DetectionMethodCommand        = "command"
	DetectionMethodFile           = "file"
	DetectionMethodBrewCask       = "brew_cask"
	DetectionMethodPackageManager = "package_manager"
)

// ValidDetectionMethods contains all valid application detection methods.
var ValidDetectionMethods = []string{
	DetectionMethodCommand,
	DetectionMethodFile,
	DetectionMethodBrewCask,
	DetectionMethodPackageManager,
}

// ApplicationConfigStrategy constants define how application configs are deployed.
const (
	AppConfigStrategySymlink  = "symlink"
	AppConfigStrategyCopy     = "copy"
	AppConfigStrategyMerge    = "merge"
	AppConfigStrategyTemplate = "template"
)

// ValidAppConfigStrategies contains all valid application config strategies.
var ValidAppConfigStrategies = []string{
	AppConfigStrategySymlink,
	AppConfigStrategyCopy,
	AppConfigStrategyMerge,
	AppConfigStrategyTemplate,
}

// FileMode constants define common file permissions.
const (
	FileModeReadOnly       = "0444"
	FileModeReadWrite      = "0644"
	FileModeReadWriteExec  = "0755"
	FileModeFullPermission = "0777"
)

// ValidFileModes contains commonly used file modes.
var ValidFileModes = []string{
	FileModeReadOnly,
	FileModeReadWrite,
	FileModeReadWriteExec,
	FileModeFullPermission,
}

// DirectoryMode constants define common directory permissions.
const (
	DirectoryModeReadOnly       = "0555"
	DirectoryModeReadWrite      = "0755"
	DirectoryModeFullPermission = "0777"
)

// ValidDirectoryModes contains commonly used directory modes.
var ValidDirectoryModes = []string{
	DirectoryModeReadOnly,
	DirectoryModeReadWrite,
	DirectoryModeFullPermission,
}

// Git constants define Git-related configurations.
const (
	GitAuthMethodNone = "none"
	GitAuthMethodSSH  = "ssh"
	GitAuthMethodPAT  = "pat"
	GitAuthMethodAuto = "auto"
)

// ValidGitAuthMethods contains all valid Git authentication methods.
var ValidGitAuthMethods = []string{
	GitAuthMethodNone,
	GitAuthMethodSSH,
	GitAuthMethodPAT,
	GitAuthMethodAuto,
}

// Provider metadata constants.
const (
	ProviderName        = "dotfiles"
	ProviderVersion     = "1.0.0"
	ProviderDescription = "Terraform provider for managing dotfiles and configuration"
)

// Resource type constants.
const (
	ResourceTypeRepository  = "dotfiles_repository"
	ResourceTypeFile        = "dotfiles_file"
	ResourceTypeSymlink     = "dotfiles_symlink"
	ResourceTypeDirectory   = "dotfiles_directory"
	ResourceTypeApplication = "dotfiles_application"
)

// Data source type constants.
const (
	DataSourceTypeSystem   = "dotfiles_system"
	DataSourceTypeFileInfo = "dotfiles_file_info"
)

// Default values for provider configuration.
const (
	DefaultStrategy           = StrategySymlink
	DefaultConflictResolution = ConflictResolutionBackup
	DefaultTargetPlatform     = PlatformAuto
	DefaultTemplateEngine     = TemplateEngineGo
	DefaultLogLevel           = LogLevelInfo
	DefaultBackupFormat       = BackupFormatTimestamped
	DefaultFileMode           = FileModeReadWrite
	DefaultDirectoryMode      = DirectoryModeReadWrite
)

// Path constants for common directories.
const (
	DefaultDotfilesDir = "dotfiles"
	DefaultBackupDir   = ".dotfiles-backups"
	ConfigDirName      = ".config"
	LocalShareDirName  = ".local/share"
)

// Environment variable names.
const (
	EnvVarDotfilesRoot = "DOTFILES_ROOT"
	EnvVarBackupDir    = "DOTFILES_BACKUP_DIR"
	EnvVarDryRun       = "DOTFILES_DRY_RUN"
	EnvVarLogLevel     = "DOTFILES_LOG_LEVEL"
	EnvVarGitToken     = "DOTFILES_GIT_TOKEN"
	EnvVarGitSSHKey    = "DOTFILES_SSH_KEY"
)

// Error codes for structured error handling.
const (
	ErrorCodeValidation = "VALIDATION_ERROR"
	ErrorCodeFileSystem = "FILESYSTEM_ERROR"
	ErrorCodeTemplate   = "TEMPLATE_ERROR"
	ErrorCodeGit        = "GIT_ERROR"
	ErrorCodePermission = "PERMISSION_ERROR"
	ErrorCodeNotFound   = "NOT_FOUND"
	ErrorCodeConflict   = "CONFLICT_ERROR"
	ErrorCodeInternal   = "INTERNAL_ERROR"
)

// Timeout constants for operations.
const (
	DefaultGitTimeout      = 30 // seconds
	DefaultFileTimeout     = 10 // seconds
	DefaultTemplateTimeout = 5  // seconds
	DefaultBackupTimeout   = 60 // seconds
)

// Size limits for operations.
const (
	MaxFileSize     = 100 * 1024 * 1024  // 100MB
	MaxTemplateSize = 10 * 1024 * 1024   // 10MB
	MaxBackupSize   = 1024 * 1024 * 1024 // 1GB
)

// Validation patterns.
const (
	// FileNamePattern matches valid file names (excluding path separators and special chars).
	FileNamePattern = `^[a-zA-Z0-9._-]+$`

	// PathPattern matches valid paths (allowing path separators and environment variables).
	PathPattern = `^[a-zA-Z0-9._/\\~${}:-]+$`

	// VersionPattern matches semantic version strings.
	VersionPattern = `^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?(?:\+([a-zA-Z0-9.-]+))?$`

	// GitURLPattern matches Git repository URLs.
	GitURLPattern = `^(https?://|git@|ssh://)`
)

// Cache configuration constants.
const (
	DefaultCacheSize     = 1000
	DefaultCacheTTL      = 300 // 5 minutes in seconds
	MaxCacheSize         = 10000
	CacheCleanupInterval = 600 // 10 minutes in seconds
)

// Concurrency constants.
const (
	DefaultMaxConcurrency = 10
	MaxConcurrency        = 50
	MinConcurrency        = 1
)
