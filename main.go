package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elev1e1n/spelunk-md/config"
	"github.com/elev1e1n/spelunk-md/generator"
	"github.com/elev1e1n/spelunk-md/prompt"
	"github.com/elev1e1n/spelunk-md/scanner"
	"github.com/elev1e1n/spelunk-md/ui"
	"github.com/spf13/cobra"
)

var version = "dev"

var (
	flagAPIKey  string
	flagModel   string
	flagPath    string
	flagDryRun  bool
	flagTimeout int
	flagForce   bool
)

type target struct {
	cmd   string
	file  string
	label string
}

var targets = []target{
	{"claude",   "CLAUDE.md",       "Claude Code"},
	{"codex",    "AGENTS.md",       "OpenAI Codex / OpenCode"},
	{"cursor",   ".cursorrules",    "Cursor"},
	{"windsurf", ".windsurfrules",  "Windsurf"},
}

func main() {
	root := &cobra.Command{
		Use:           "spelunk-md",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagAPIKey != "" {
				return handleAPIKey()
			}
			helpTargets := make([]ui.HelpTarget, len(targets))
			for i, t := range targets {
				helpTargets[i] = ui.HelpTarget{Cmd: t.cmd, File: t.file, Label: t.label}
			}
			ui.Help(version, helpTargets)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", `OpenRouter API key (or "clear" to remove)`)
	root.PersistentFlags().StringVar(&flagModel, "model", config.DefaultModel, "OpenRouter model")
	root.PersistentFlags().StringVar(&flagPath, "path", ".", "Project root path")
	root.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Print prompt without calling API")
	root.PersistentFlags().IntVar(&flagTimeout, "timeout", 120, "Request timeout in seconds")
	root.PersistentFlags().BoolVar(&flagForce, "force", false, "Overwrite without confirmation")

	for _, t := range targets {
		t := t
		root.AddCommand(&cobra.Command{
			Use:           t.cmd,
			Short:         fmt.Sprintf("Generate %s for %s", t.file, t.label),
			SilenceUsage:  true,
			SilenceErrors: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return generate([]target{t})
			},
		})
	}

	root.AddCommand(&cobra.Command{
		Use:           "all",
		Short:         "Generate all context files in parallel",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generate(targets)
		},
	})

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleAPIKey() error {
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
	return nil
}

func generate(ts []target) error {
	if flagAPIKey != "" {
		if err := handleAPIKey(); err != nil {
			return err
		}
		if flagPath == "." && !flagDryRun {
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

	ui.Header("spelunk-md", root)

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
		ProjectName: filepath.Base(root),
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

	// Resolve output paths; ask confirmation for existing files.
	type resolved struct {
		target
		outPath string
	}
	var work []resolved
	for _, t := range ts {
		outPath := filepath.Join(root, t.file)
		if !flagForce {
			if _, err := os.Stat(outPath); err == nil {
				ui.Confirm(t.file)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Println()
					continue
				}
			}
		}
		work = append(work, resolved{t, outPath})
	}
	if len(work) == 0 {
		return nil
	}

	ui.Divider()
	spin = ui.NewModelSpinner(flagModel)
	spin.Start()

	type result struct {
		r       resolved
		content string
		err     error
	}
	results := make([]result, len(work))
	var wg sync.WaitGroup
	for i, r := range work {
		wg.Add(1)
		go func(i int, r resolved) {
			defer wg.Done()
			content, err := generator.Generate(apiKey, flagModel, p, flagTimeout)
			results[i] = result{r, content, err}
		}(i, r)
	}
	wg.Wait()
	spin.Stop()

	for _, res := range results {
		if res.err != nil {
			ui.Fail(res.r.file + ": " + res.err.Error())
			continue
		}
		if err := generator.WriteFile(res.r.outPath, res.content); err != nil {
			ui.Fail(res.r.file + ": " + err.Error())
			continue
		}
		size := fmt.Sprintf("%.1f KB", float64(len(res.content))/1024)
		ui.Success(res.r.file, res.r.outPath+"  "+size)
	}
	fmt.Println()
	return nil
}

func stackLabel(s *scanner.Stack) string {
	parts := append(s.Languages, s.Frameworks...)
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
