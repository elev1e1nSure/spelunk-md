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
	fmt.Printf("\n%s%s%s  %s%s%s\n\n",
		bold+teal, name, reset,
		faint, path, reset,
	)
}

// Step prints "◆  label   value" with space-aligned columns, no dots.
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
	fmt.Printf("\n%s✓%s  %s%s%s  %s%s%s\n\n",
		sage, reset,
		bold, filename, reset,
		faint, meta, reset,
	)
}

func Fail(msg string) {
	fmt.Printf("\n%s✗%s  %s\n\n", coral, reset, msg)
}

func KeySaved(msg string) {
	fmt.Printf("\n%s●%s  %s%s%s\n\n", teal, reset, bold, msg, reset)
}

func DryRun(prompt string) {
	fmt.Printf("%sdry run%s\n\n", faint, reset)
	fmt.Println(prompt)
	fmt.Println()
}

type Spinner struct {
	label string
	stop  chan struct{}
	done  chan struct{}
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NewSpinner(label string) *Spinner {
	return &Spinner{label: label, stop: make(chan struct{}), done: make(chan struct{})}
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
				fmt.Printf("\r%s%s%s  %s%s%s",
					lavender, spinFrames[i%len(spinFrames)], reset,
					faint, s.label, reset,
				)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}
