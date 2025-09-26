# Implementation Plan for Terraform Dotfiles Provider

## Introduction

This implementation plan addresses the findings from the code review of the `terraform-provider-dotfiles`. The goal is to resolve code smells, fix issues, improve architecture, and implement enhancements to achieve a stable v1.0 release. The plan is divided into phases based on priority:

- **Phase 1: Critical Fixes** (High Priority): Complete stubs, fix duplication, and ensure core functionality works.
- **Phase 2: Refinements and Validation** (Medium Priority): Enhance validation, error handling, and idempotency.
- **Phase 3: Architectural Improvements** (Medium Priority): Refactor for better modularity and scalability.
- **Phase 4: Testing and Polish** (Medium Priority): Expand tests and documentation.
- **Phase 5: New Features and Enhancements** (Low Priority): Add advanced capabilities.

Estimated total effort: 4-6 weeks (assuming 1 developer, 40h/week). Milestones include testable alphas after each phase.

## Phase 1: Critical Fixes (1-2 weeks)

Focus: Eliminate TODOs and ensure basic operations (Create/Read/Update/Delete) are functional across resources.

### Tasks
1. **Complete Application Resource Implementation**
   - Implement file/symlink deployment based on `config_mappings` in Create/Update.
   - Add recursive handling for app-specific dirs (e.g., copy entire config folder).
   - Handle skip/warn behaviors fully in Read (drift detection).
   - *Files*: `internal/provider/application_resource.go` (lines ~168-217, add ~100 lines).
   - *Dependencies*: Platform ops (symlink/copy).
   - *Effort*: 4h.

2. **Implement Directory Resource Delete and Recursion**
   - Add recursive directory removal in Delete with safety checks (e.g., skip if contains system files).
   - Enhance Read to verify contents and permissions.
   - Update to support nested structures.
   - *Files*: `internal/provider/directory_resource.go` (lines 92-119, add ~50 lines).
   - *Dependencies*: Fileops for recursive ops.
   - *Effort*: 3h.

3. **Finish Symlink Resource Platform Support**
   - Implement Windows symlink creation (use `syscall` or admin checks).
   - Add Read to verify link targets.
   - *Files*: `internal/provider/symlink_resource.go`, `internal/platform/windows.go` (CreateSymlink).
   - *Dependencies*: Platform interface.
   - *Effort*: 4h.

4. **Complete Template Rendering in File Resource**
   - Full Handlebars support in `internal/template/enhanced_engines.go`.
   - Validate `template_vars` in schema (ensure string keys).
   - Add error handling for render failures.
   - *Files*: `internal/provider/file_resource.go` (Create lines ~100-300), `internal/template/enhanced_engines.go`.
   - *Dependencies*: Template engine deps.
   - *Effort*: 6h.

5. **Remove Config Duplication**
   - Centralize defaults and path expansion in `config.go` Validate.
   - Refactor Configure to call Validate early.
   - Deprecate overlapping `backup_strategy` fields or merge.
   - *Files*: `internal/provider/provider.go` (lines 106-246), `internal/provider/config.go` (lines 29-126).
   - *Dependencies*: None.
   - *Effort*: 4h.

### Testing
- Update unit tests for new logic (e.g., app deployment mocks).
- Run integration tests on all platforms (use Docker for Linux/Windows).

### Milestone
- All resources CRUD functional; `terraform apply` deploys basic dotfiles without errors.

## Phase 2: Refinements and Validation (1 week)

Focus: Improve reliability and user experience.

### Tasks
1. **Add Schema Validators**
   - Path validators for `source_path`/`target_path` (absolute, no invalid chars).
   - Octal mode validator for `file_mode`.
   - Environment var support in paths.
   - *Files*: All resource schemas (`*_resource.go`).
   - *Dependencies*: Framework validators.
   - *Effort*: 4h.

2. **Enhance Error Handling and Idempotency**
   - Wrap errors with context (`fmt.Errorf` + `%w`).
   - Add retry for Git ops (e.g., 3x for network issues).
   - In Read, compare rendered content/hashes for templates.
   - Ensure dry-run skips all I/O.
   - *Files*: Resources (e.g., `file_resource.go` Create/Read), `internal/git/operations.go`.
   - *Dependencies*: Phase 1.
   - *Effort*: 6h.

3. **Runtime Checks**
   - Validate `dotfiles_root` writable in Configure.
   - Check source existence pre-apply.
   - *Files*: `internal/provider/config.go`, resources' ValidateResource.
   - *Dependencies*: None.
   - *Effort*: 3h.

### Testing
- Add validation error tests.
- Fuzz paths/templates.

### Milestone
- No runtime panics; plan shows accurate diffs.

## Phase 3: Architectural Improvements (1 week)

Focus: Better structure for maintainability.

### Tasks
1. **Introduce Service Layer**
   - `BackupService`, `TemplateService` interfaces.
   - Inject into client/resources.
   - *Files*: New `internal/services/` dir; refactor `fileops/`, `template/`.
   - *Dependencies*: Phase 1.
   - *Effort*: 8h.

2. **Constants and Enums**
   - Define `const` for strategies/platforms.
   - Use enum attributes in schema.
   - *Files*: `internal/provider/schema.go` (new), update all.
   - *Dependencies*: None.
   - *Effort*: 3h.

3. **Caching and Concurrency**
   - In-memory cache for hashes/detections.
   - Use `errgroup` for parallel file ops.
   - Mutex for shared client state.
   - *Files*: `internal/provider/client.go`, resources.
   - *Dependencies*: Phase 2.
   - *Effort*: 5h.

4. **Git Enhancements**
   - Auth for private repos (tokens).
   - Submodule support.
   - *Files*: `internal/git/operations.go`, `internal/provider/repository_resource.go`.
   - *Dependencies*: `go-git` updates.
   - *Effort*: 4h.

### Testing
- Integration tests for services.
- Concurrency tests (e.g., parallel applies).

### Milestone
- Refactored code passes all tests; improved perf (e.g., 20% faster applies).

## Phase 4: Testing and Polish (3-5 days)

Focus: Ensure quality.

### Tasks
1. **Expand Tests**
   - Complete E2E (`end_to_end_integration_test.go`).
   - Cross-platform CI matrix.
   - Fuzz/edge cases (empty vars, large files).
   - *Files*: All `*_test.go`.
   - *Dependencies*: All prior phases.
   - *Effort*: 6h.

2. **Documentation**
   - Update README with new features.
   - Generate schema docs.
   - Add migration guide for config changes.
   - *Files*: `README.md`, `docs/*`.
   - *Dependencies*: Phase 3.
   - *Effort*: 4h.

3. **Linting and Tools**
   - Run `go vet`, `staticcheck`.
   - Add pre-commit hooks.
   - *Files*: `.golangci.yml` (new).
   - *Dependencies*: None.
   - *Effort*: 2h.

### Testing
- Full suite: 100% coverage goal.
- Manual: Deploy sample dotfiles on macOS/Linux.

### Milestone
- Tests pass; docs complete. Beta release candidate.

## Phase 5: New Features and Enhancements (1-2 weeks, Optional for v1.0)

Focus: Future-proofing.

### Tasks
1. **Advanced Resources**
   - `dotfiles_package` for installs (Homebrew/Apt).
   - *Files*: New resource in `internal/provider/`.
   - *Effort*: 8h.

2. **Functions and Ephemerals**
   - `template_render` function.
   - Ephemeral for temp scripts.
   - *Files*: `internal/provider/provider.go`, new funcs.
   - *Effort*: 6h.

3. **Recovery and Monitoring**
   - Full restore TF generation.
   - Backup stats attrs.
   - *Files*: `internal/fileops/enhanced_backup.go`.
   - *Effort*: 5h.

4. **Security/Perf**
   - Backup encryption.
   - OTEL integration.
   - *Files*: Services, client.
   - *Effort*: 4h.

### Testing
- New feature tests.

### Milestone
- v1.0 release: Feature-complete, published to registry.

## Risks and Dependencies
- **Platform Testing**: Need access to Windows/macOS VMs.
- **External Deps**: `go-git` updates may break Git ops.
- **Terraform Framework**: Align with latest versions (v1.5+).
- **Review**: Peer review each phase.

## Timeline
- Week 1: Phase 1
- Week 2: Phase 2
- Week 3: Phase 3
- Week 4: Phase 4 + Release
- Optional: Phase 5 post-v1.0

Track progress via GitHub issues/PRs. Use this plan as a living document.
