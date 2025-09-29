package migration

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// ValidationIssue represents a compatibility issue found in a configuration.
type ValidationIssue struct {
	LineNumber  int
	Description string
	Suggestion  string
}

// MigrateConfigFile migrates a Terraform configuration file from the old format to the new format.
func MigrateConfigFile(inputFile, outputFile string) error {
	input, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		if cerr := input.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if cerr := output.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return migrateConfig(input, output)
}

// ValidateConfigFile checks a Terraform configuration file for compatibility issues.
func ValidateConfigFile(inputFile string) ([]ValidationIssue, error) {
	input, err := os.Open(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		_ = input.Close() // Best effort close, errors are not critical for validation
	}()

	return validateConfig(input)
}

// migrateConfig performs the actual migration from input to output.
func migrateConfig(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer func() {
		_ = writer.Flush() // Best effort flush, errors handled elsewhere
	}()

	var currentResource *ResourceContext
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// Check if we're starting a new dotfiles_file resource
		if match := fileResourceRegex.FindStringSubmatch(line); match != nil {
			currentResource = &ResourceContext{
				ResourceType: "dotfiles_file",
				ResourceName: match[1],
				StartLine:    lineNumber,
				Lines:        []string{line},
			}
			continue
		}

		// If we're inside a resource, collect lines
		if currentResource != nil {
			currentResource.Lines = append(currentResource.Lines, line)

			// Check for strategy field
			if match := strategyFieldRegex.FindStringSubmatch(line); match != nil {
				currentResource.Strategy = strings.Trim(match[1], `"`)
			}

			// Check if we've reached the end of the resource
			if strings.TrimSpace(line) == "}" && isResourceEnd(currentResource.Lines) {
				// Process the resource
				migratedLines, err := migrateResource(currentResource)
				if err != nil {
					return fmt.Errorf("failed to migrate resource %s at line %d: %w",
						currentResource.ResourceName, currentResource.StartLine, err)
				}

				// Write migrated resource
				for _, migratedLine := range migratedLines {
					if _, err := writer.WriteString(migratedLine + "\n"); err != nil {
						return err
					}
				}

				currentResource = nil
				continue
			}
		}

		// If not inside a resource or resource doesn't need migration, write as-is
		if currentResource == nil {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// validateConfig checks for compatibility issues without modifying the config.
func validateConfig(input io.Reader) ([]ValidationIssue, error) {
	scanner := bufio.NewScanner(input)
	var issues []ValidationIssue
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// Check for dotfiles_file resources with strategy field
		if fileResourceRegex.MatchString(line) {
			resourceName := fileResourceRegex.FindStringSubmatch(line)[1]

			// Look ahead to find strategy field in this resource
			resourceLines := []string{line}
			braceCount := strings.Count(line, "{") - strings.Count(line, "}")

			for braceCount > 0 && scanner.Scan() {
				lineNumber++
				nextLine := scanner.Text()
				resourceLines = append(resourceLines, nextLine)
				braceCount += strings.Count(nextLine, "{") - strings.Count(nextLine, "}")

				// Check for strategy field
				if match := strategyFieldRegex.FindStringSubmatch(nextLine); match != nil {
					strategy := strings.Trim(match[1], `"`)
					issue := ValidationIssue{
						LineNumber:  lineNumber,
						Description: fmt.Sprintf("Resource '%s' uses deprecated strategy field with value '%s'", resourceName, strategy),
					}

					switch strategy {
					case "symlink":
						issue.Suggestion = "Convert to dotfiles_symlink resource"
					case "copy":
						issue.Suggestion = "Remove strategy field (dotfiles_file defaults to copy)"
					case "template":
						issue.Suggestion = "Remove strategy field and set is_template = true"
					default:
						issue.Suggestion = "Consider using dotfiles_application for complex strategies"
					}

					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, scanner.Err()
}

// ResourceContext holds information about a resource being processed.
type ResourceContext struct {
	ResourceType string
	ResourceName string
	Strategy     string
	StartLine    int
	Lines        []string
}

// migrateResource converts a dotfiles_file resource to the appropriate new format.
func migrateResource(ctx *ResourceContext) ([]string, error) {
	if ctx.Strategy == "" {
		// No strategy field, just remove any strategy references and keep as dotfiles_file
		return removeStrategyField(ctx.Lines), nil
	}

	switch ctx.Strategy {
	case "symlink":
		return convertToSymlinkResource(ctx)
	case "copy":
		// Convert to dotfiles_file without strategy field
		return removeStrategyField(ctx.Lines), nil
	case "template":
		return convertToTemplateFile(ctx)
	default:
		// Unknown strategy, add comment and keep as dotfiles_file
		lines := removeStrategyField(ctx.Lines)
		lines[0] = "# MIGRATION NOTE: Unknown strategy '" + ctx.Strategy + "' converted to copy operation\n" + lines[0]
		return lines, nil
	}
}

// removeStrategyField removes strategy field lines from the resource.
func removeStrategyField(lines []string) []string {
	var result []string
	for _, line := range lines {
		if !strategyFieldRegex.MatchString(line) {
			result = append(result, line)
		}
	}
	return result
}

// convertToSymlinkResource converts dotfiles_file with strategy=symlink to dotfiles_symlink.
func convertToSymlinkResource(ctx *ResourceContext) ([]string, error) {
	result := make([]string, 0, len(ctx.Lines)+1)

	// Add migration comment
	result = append(result, "# MIGRATED: dotfiles_file with strategy=symlink → dotfiles_symlink")

	for i, line := range ctx.Lines {
		if i == 0 {
			// Replace resource type
			line = strings.Replace(line, "dotfiles_file", "dotfiles_symlink", 1)
		} else if strategyFieldRegex.MatchString(line) {
			// Skip strategy field
			continue
		}
		result = append(result, line)
	}

	return result, nil
}

// convertToTemplateFile converts dotfiles_file with strategy=template to dotfiles_file with is_template=true.
func convertToTemplateFile(ctx *ResourceContext) ([]string, error) {
	result := make([]string, 0, len(ctx.Lines)+2)
	var hasIsTemplate bool

	// Add migration comment
	result = append(result, "# MIGRATED: strategy=template → is_template=true")

	for _, line := range ctx.Lines {
		if strategyFieldRegex.MatchString(line) {
			// Replace strategy field with is_template
			indentation := getIndentation(line)
			result = append(result, indentation+"is_template = true")
			continue
		}

		if strings.Contains(line, "is_template") {
			hasIsTemplate = true
		}

		result = append(result, line)
	}

	// If is_template wasn't already set, we've added it above
	if hasIsTemplate {
		// Add note that there might be a conflict
		result[1] += " # NOTE: Check for is_template conflicts"
	}

	return result, nil
}

// isResourceEnd checks if we've reached the end of a resource block.
func isResourceEnd(lines []string) bool {
	braceCount := 0
	for _, line := range lines {
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
	}
	return braceCount == 0
}

// getIndentation returns the indentation (whitespace) from the beginning of a line.
func getIndentation(line string) string {
	for i, char := range line {
		if char != ' ' && char != '\t' {
			return line[:i]
		}
	}
	return line
}

// Regular expressions for parsing Terraform configuration.
var (
	fileResourceRegex  = regexp.MustCompile(`resource\s+"dotfiles_file"\s+"([^"]+)"`)
	strategyFieldRegex = regexp.MustCompile(`\s*strategy\s*=\s*"([^"]*)"`)
)
