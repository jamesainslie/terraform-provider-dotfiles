package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jamesainslie/terraform-provider-dotfiles/internal/migration"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		if len(os.Args) < 4 {
			printUsage()
			os.Exit(1)
		}
		inputFile := os.Args[2]
		outputFile := os.Args[3]

		if err := migration.MigrateConfigFile(inputFile, outputFile); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

		fmt.Printf("Configuration migrated successfully!\n")
		fmt.Printf("Input:  %s\n", inputFile)
		fmt.Printf("Output: %s\n", outputFile)
		fmt.Printf("\nPlease review the output file and test the configuration before applying.\n")

	case "validate":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		inputFile := os.Args[2]

		issues, err := migration.ValidateConfigFile(inputFile)
		if err != nil {
			log.Fatalf("Validation failed: %v", err)
		}

		if len(issues) == 0 {
			fmt.Printf("✅ Configuration %s is compatible with the new architecture.\n", inputFile)
		} else {
			fmt.Printf("⚠️  Found %d compatibility issue(s) in %s:\n\n", len(issues), inputFile)
			for i, issue := range issues {
				fmt.Printf("%d. %s\n", i+1, issue.Description)
				if issue.Suggestion != "" {
					fmt.Printf("   Suggestion: %s\n", issue.Suggestion)
				}
				fmt.Printf("\n")
			}
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`terraform-provider-dotfiles Migration Tool

This tool helps migrate Terraform configurations to the new explicit resource architecture
where the strategy field has been removed from dotfiles_file resource.

Usage:
  %s migrate <input.tf> <output.tf>     Migrate configuration file
  %s validate <input.tf>                Check for compatibility issues

Examples:
  %s migrate main.tf main-migrated.tf
  %s validate main.tf

Migration Details:
- dotfiles_file resources with strategy="symlink" → dotfiles_symlink resources
- dotfiles_file resources with strategy="copy" → dotfiles_file resources (strategy field removed)
- dotfiles_file resources with strategy="template" → dotfiles_file resources (converted to is_template=true)
- Complex patterns with multiple strategies → dotfiles_application resources

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
