package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elev1e1n/claude-md-gen/config"
	"github.com/elev1e1n/claude-md-gen/generator"
	"github.com/elev1e1n/claude-md-gen/prompt"
	"github.com/elev1e1n/claude-md-gen/scanner"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey string
	flagModel  string
	flagOutput string
	flagPath   string
	flagDryRun bool
)

func main() {
	root := &cobra.Command{
		Use:   "claude-md-gen",
		Short: "Generate CLAUDE.md for any codebase using AI",
		Long: `Scans your project (files, stack, git history) and generates
a tailored CLAUDE.md using an AI model via OpenRouter.`,
		RunE: run,
	}

	root.Flags().StringVar(&flagAPIKey, "api-key", "", `OpenRouter API key. Use "clear" to remove saved key`)
	root.Flags().StringVar(&flagModel, "model", config.DefaultModel, "OpenRouter model to use")
	root.Flags().StringVar(&flagOutput, "output", "CLAUDE.md", "Output file path")
	root.Flags().StringVar(&flagPath, "path", ".", "Project root path")
	root.Flags().BoolVar(&flagDryRun, "dry-run", false, "Print the prompt without calling the API")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Handle api-key flag first
	if flagAPIKey != "" {
		if flagAPIKey == "clear" {
			if err := config.DeleteAPIKey(); err != nil {
				return err
			}
			fmt.Println("API key removed.")
			return nil
		}
		if err := config.SetAPIKey(flagAPIKey); err != nil {
			return err
		}
		fmt.Println("API key saved to system keyring.")
		// Continue to generate unless that's all they wanted
		// (if no other meaningful context, just exit)
		if flagPath == "." && !flagDryRun {
			fmt.Println("Run again without --api-key to generate CLAUDE.md")
			return nil
		}
	}

	// Resolve project root
	root, err := filepath.Abs(flagPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", root)
	}

	projectName := filepath.Base(root)
	fmt.Printf("→ Scanning %s\n", root)

	// Scan
	fmt.Println("  [1/3] Reading files...")
	tree, err := scanner.ScanFiles(root)
	if err != nil {
		return err
	}
	fmt.Printf("        found %d files\n", len(tree.Entries))

	fmt.Println("  [2/3] Detecting stack...")
	stack := scanner.DetectStack(root, tree.Entries)
	if len(stack.Languages) > 0 {
		fmt.Printf("        languages: %v\n", stack.Languages)
	}
	if len(stack.Frameworks) > 0 {
		fmt.Printf("        frameworks: %v\n", stack.Frameworks)
	}

	fmt.Println("  [3/3] Reading git history...")
	git := scanner.ScanGit(root)
	if git.IsRepo {
		fmt.Printf("        branch: %s\n", git.Branch)
	} else {
		fmt.Println("        (not a git repo)")
	}

	// Build prompt
	ctx := &prompt.Context{
		ProjectName: projectName,
		Tree:        tree,
		Stack:       stack,
		Git:         git,
	}
	p := prompt.Build(ctx)

	if flagDryRun {
		fmt.Println("\n─── DRY RUN PROMPT ───────────────────────────────────────")
		fmt.Println(p)
		fmt.Println("──────────────────────────────────────────────────────────")
		return nil
	}

	// Get API key
	apiKey, err := config.GetAPIKey()
	if err != nil {
		return err
	}

	// Generate
	fmt.Printf("\n→ Calling %s via OpenRouter...\n", flagModel)
	content, err := generator.Generate(apiKey, flagModel, p)
	if err != nil {
		return err
	}

	// Write output
	outputPath := flagOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(root, outputPath)
	}

	if err := generator.WriteFile(outputPath, content); err != nil {
		return err
	}

	fmt.Printf("✓ CLAUDE.md written to %s\n", outputPath)
	return nil
}
