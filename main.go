package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elev1e1n/claude-md-gen/config"
	"github.com/elev1e1n/claude-md-gen/generator"
	"github.com/elev1e1n/claude-md-gen/prompt"
	"github.com/elev1e1n/claude-md-gen/scanner"
	"github.com/elev1e1n/claude-md-gen/ui"
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
		Use:   "spelunk",
		Short: "Generate CLAUDE.md for any codebase using AI",
		Long:  `Scans your project (files, stack, git history) and generates a tailored CLAUDE.md via OpenRouter.`,
		RunE:  run,
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
	if flagAPIKey != "" {
		if flagAPIKey == "clear" {
			if err := config.DeleteAPIKey(); err != nil {
				return err
			}
			ui.KeySaved("API key removed from keyring")
			return nil
		}
		if err := config.SetAPIKey(flagAPIKey); err != nil {
			return err
		}
		ui.KeySaved("API key saved to keyring")
		if flagPath == "." && !flagDryRun {
			fmt.Println("  Run again without --api-key to generate CLAUDE.md")
			return nil
		}
	}

	root, err := filepath.Abs(flagPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", root)
	}

	projectName := filepath.Base(root)
	ui.Header("spelunk", root)

	spin := ui.NewSpinner("reading files")
	spin.Start()
	tree, err := scanner.ScanFiles(root)
	spin.Stop()
	if err != nil {
		return err
	}
	ui.Step("files", fmt.Sprintf("%d", len(tree.Entries)))

	spin = ui.NewSpinner("detecting stack")
	spin.Start()
	stack := scanner.DetectStack(root, tree.Entries)
	spin.Stop()
	ui.Step("stack", stackLabel(stack))

	spin = ui.NewSpinner("reading git")
	spin.Start()
	git := scanner.ScanGit(root)
	spin.Stop()
	ui.Step("git", gitLabel(git))

	ctx := &prompt.Context{
		ProjectName: projectName,
		Tree:        tree,
		Stack:       stack,
		Git:         git,
	}
	p := prompt.Build(ctx)

	if flagDryRun {
		ui.DryRun(p)
		return nil
	}

	apiKey, err := config.GetAPIKey()
	if err != nil {
		return err
	}

	ui.Divider()
	spin = ui.NewSpinner(flagModel)
	spin.Start()
	content, err := generator.Generate(apiKey, flagModel, p)
	spin.Stop()
	if err != nil {
		return err
	}

	outputPath := flagOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(root, outputPath)
	}
	if err := generator.WriteFile(outputPath, content); err != nil {
		return err
	}

	size := fmt.Sprintf("%.1f KB", float64(len(content))/1024)
	ui.Success(filepath.Base(outputPath), outputPath+"  "+size)
	return nil
}

func stackLabel(s *scanner.Stack) string {
	var parts []string
	parts = append(parts, s.Languages...)
	parts = append(parts, s.Frameworks...)
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "  ")
}

func gitLabel(g *scanner.GitInfo) string {
	if !g.IsRepo {
		return "not a git repo"
	}
	commitCount := 0
	if g.RecentCommits != "" {
		commitCount = len(strings.Split(strings.TrimSpace(g.RecentCommits), "\n"))
	}
	if commitCount > 0 {
		return fmt.Sprintf("%s  (%d commits)", g.Branch, commitCount)
	}
	return g.Branch
}
