package ui

import (
	"fmt"
	"strings"
	"time"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"
	faint = "\033[2m"

	teal     = "\033[38;5;116m"
	lavender = "\033[38;5;183m"
	chalk    = "\033[97m"
	sage     = "\033[38;5;114m"
	coral    = "\033[38;5;210m"
)

func init() { enableANSI() }

func Header(name, path string) {
	fmt.Printf("\n%s%s%s %s%s%s\n\n",
		bold+teal, name, reset,
		faint, path, reset,
	)
}

func Step(label, value string) {
	const width = 8
	pad := width - len(label)
	if pad < 1 {
		pad = 1
	}
	fmt.Printf("%s◆%s  %s%s%s%s  %s%s%s\n",
		teal, reset,
		bold, label, reset,
		strings.Repeat(" ", pad),
		chalk, value, reset,
	)
}

func Divider() { fmt.Println() }

func Success(filename, meta string) {
	fmt.Printf("%s✓%s  %s%-20s%s  %s%s%s\n",
		sage, reset,
		bold, filename, reset,
		faint, meta, reset,
	)
}

func Fail(msg string) {
	fmt.Printf("%s✗%s  %s\n", coral, reset, msg)
}

func KeySaved(msg string) {
	fmt.Printf("\n%s●%s  %s%s%s\n\n", teal, reset, bold, msg, reset)
}

func Confirm(filename string) {
	fmt.Printf("\n%s◆%s  %s%s%s  %sexists — overwrite?%s %s[y/N]%s ",
		coral, reset,
		bold, filename, reset,
		faint, reset,
		chalk, reset,
	)
}

// AskOutput prints a styled prompt and reads a file path from stdin.
func AskOutput() string {
	fmt.Printf("\n%s◆%s  %soutput%s  %sfile path:%s ",
		teal, reset,
		bold, reset,
		faint, reset,
	)
	var input string
	fmt.Scanln(&input)
	fmt.Println()
	return strings.TrimSpace(input)
}

func DryRun(prompt string) {
	fmt.Printf("%sdry run%s\n\n", faint, reset)
	fmt.Println(prompt)
	fmt.Println()
}

// HelpTarget describes one subcommand entry for the help screen.
type HelpTarget struct {
	Cmd   string
	File  string
	Label string
}

func Help(version string, targets []HelpTarget) {
	fmt.Printf("\n%sspelunk-md%s  %s%s%s\n", bold+teal, reset, faint, version, reset)
	fmt.Printf("%sGenerates context files for AI coding tools via OpenRouter.%s\n\n", faint, reset)

	for _, t := range targets {
		fmt.Printf("  %s%-10s%s  %s%-20s%s  %s%s%s\n",
			bold+chalk, t.Cmd, reset,
			teal, t.File, reset,
			faint, t.Label, reset,
		)
	}
	fmt.Printf("  %s%-10s%s  %s%-20s%s  %sall targets, parallel%s\n",
		bold+chalk, "all", reset,
		teal, "———", reset,
		faint, reset,
	)
	fmt.Printf("  %s%-10s%s  %s%-20s%s  %sany file, any path%s\n\n",
		bold+chalk, "custom", reset,
		teal, "<your file>", reset,
		faint, reset,
	)

	flags := [][2]string{
		{"--api-key <key>", "save OpenRouter API key to keyring"},
		{"--api-key clear", "remove saved key"},
		{"--model <model>", "override model  (default: deepseek/deepseek-v4-flash)"},
		{"--path <path>", "project root  (default: .)"},
		{"--dry-run", "print prompt, no API call"},
		{"--force", "skip overwrite confirmation"},
		{"--timeout <sec>", "request timeout  (default: 120s)"},
		{"--version", "show version"},
	}
	for _, f := range flags {
		fmt.Printf("  %s%-24s%s  %s%s%s\n", chalk, f[0], reset, faint, f[1], reset)
	}
	fmt.Println()
}

type Spinner struct {
	label   string
	isModel bool
	stop    chan struct{}
	done    chan struct{}
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NewSpinner(label string) *Spinner {
	return &Spinner{label: label, stop: make(chan struct{}), done: make(chan struct{})}
}

func NewModelSpinner(model string) *Spinner {
	displayName := model
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		displayName = model[idx+1:]
	}
	return &Spinner{
		label:   displayName,
		isModel: true,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				fmt.Printf("\r\033[K")
				return
			default:
				if s.isModel {
					fmt.Printf("\r%s%s%s  %s%s%s  %s%s%s",
						lavender, spinFrames[i%len(spinFrames)], reset,
						chalk, s.label, reset,
						faint, "thinking", reset,
					)
				} else {
					fmt.Printf("\r%s%s%s  %s%s%s",
						lavender, spinFrames[i%len(spinFrames)], reset,
						faint, s.label, reset,
					)
				}
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}
