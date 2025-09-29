package main

import (
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

		log.Println("Configuration migrated successfully!")
		log.Printf("Input:  %s", inputFile)
		log.Printf("Output: %s", outputFile)
		log.Println("\nPlease review the output file and test the configuration before applying.")

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
			log.Printf("✓ Configuration %s is compatible with the new architecture.", inputFile)
		} else {
			log.Printf("⚠ Found %d compatibility issue(s) in %s:\n", len(issues), inputFile)
			for i, issue := range issues {
				log.Printf("%d. %s", i+1, issue.Description)
				if issue.Suggestion != "" {
					log.Printf("   Suggestion: %s", issue.Suggestion)
				}
			}
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	log.Printf(`terraform-provider-dotfiles Migration Tool

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
