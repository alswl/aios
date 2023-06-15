// Package term provides information about the terminal that the current process is connected to (if any),
// for example measuring the dimensions of the terminal and inspecting whether it's safe to output color.
package aios

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
	xterm "golang.org/x/term"
)

// GHTerm represents information about the terminal that a process is connected to.
type GHTerm struct {
	in           *os.File
	out          *os.File
	errOut       *os.File
	isTTY        bool
	colorEnabled bool
	is256enabled bool
	hasTrueColor bool
	width        int
	widthPercent int
}

// FromEnv initializes a GHTerm from [os.Stdout] and environment variables:
//   - GH_FORCE_TTY
//   - NO_COLOR
//   - CLICOLOR
//   - CLICOLOR_FORCE
//   - TERM
//   - COLORTERM
func FromEnv() GHTerm {
	var stdoutIsTTY bool
	var isColorEnabled bool
	var termWidthOverride int
	var termWidthPercentage int

	spec := os.Getenv("GH_FORCE_TTY")
	if spec != "" {
		stdoutIsTTY = true
		isColorEnabled = !IsColorDisabled()

		if w, err := strconv.Atoi(spec); err == nil {
			termWidthOverride = w
		} else if strings.HasSuffix(spec, "%") {
			if p, err := strconv.Atoi(spec[:len(spec)-1]); err == nil {
				termWidthPercentage = p
			}
		}
	} else {
		stdoutIsTTY = IsTerminal(os.Stdout)
		isColorEnabled = IsColorForced() || (!IsColorDisabled() && stdoutIsTTY)
	}

	isVirtualTerminal := false
	if stdoutIsTTY {
		if err := enableVirtualTerminalProcessing(os.Stdout); err == nil {
			isVirtualTerminal = true
		}
	}

	return GHTerm{
		in:           os.Stdin,
		out:          os.Stdout,
		errOut:       os.Stderr,
		isTTY:        stdoutIsTTY,
		colorEnabled: isColorEnabled,
		is256enabled: isVirtualTerminal || is256ColorSupported(),
		hasTrueColor: isVirtualTerminal || isTrueColorSupported(),
		width:        termWidthOverride,
		widthPercent: termWidthPercentage,
	}
}

// In is the reader reading from standard input.
func (t GHTerm) In() io.Reader {
	return t.in
}

// Out is the writer writing to standard output.
func (t GHTerm) Out() io.Writer {
	return t.out
}

// ErrOut is the writer writing to standard error.
func (t GHTerm) ErrOut() io.Writer {
	return t.errOut
}

// IsTerminalOutput returns true if standard output is connected to a terminal.
func (t GHTerm) IsTerminalOutput() bool {
	return t.isTTY
}

// IsColorEnabled reports whether it's safe to output ANSI color sequences, depending on IsTerminalOutput
// and environment variables.
func (t GHTerm) IsColorEnabled() bool {
	return t.colorEnabled
}

// Is256ColorSupported reports whether the terminal advertises ANSI 256 color codes.
func (t GHTerm) Is256ColorSupported() bool {
	return t.is256enabled
}

// IsTrueColorSupported reports whether the terminal advertises support for ANSI true color sequences.
func (t GHTerm) IsTrueColorSupported() bool {
	return t.hasTrueColor
}

// Size returns the width and height of the terminal that the current process is attached to.
// In case of errors, the numeric values returned are -1.
func (t GHTerm) Size() (int, int, error) {
	if t.width > 0 {
		return t.width, -1, nil
	}

	ttyOut := t.out
	if ttyOut == nil || !IsTerminal(ttyOut) {
		if f, err := openTTY(); err == nil {
			defer f.Close()
			ttyOut = f
		} else {
			return -1, -1, err
		}
	}

	width, height, err := terminalSize(ttyOut)
	if err == nil && t.widthPercent > 0 {
		return int(float64(width) * float64(t.widthPercent) / 100), height, nil
	}

	return width, height, err
}

// Theme returns the theme of the terminal by analyzing the background color of the terminal.
func (t GHTerm) Theme() string {
	if !t.IsColorEnabled() {
		return "none"
	}
	if termenv.HasDarkBackground() {
		return "dark"
	}
	return "light"
}

// IsTerminal reports whether a file descriptor is connected to a terminal.
func IsTerminal(f *os.File) bool {
	return xterm.IsTerminal(int(f.Fd()))
}

func terminalSize(f *os.File) (int, int, error) {
	return xterm.GetSize(int(f.Fd()))
}

// IsColorDisabled returns true if environment variables NO_COLOR or CLICOLOR prohibit usage of color codes
// in terminal output.
func IsColorDisabled() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0"
}

// IsColorForced returns true if environment variable CLICOLOR_FORCE is set to force colored terminal output.
func IsColorForced() bool {
	return os.Getenv("CLICOLOR_FORCE") != "" && os.Getenv("CLICOLOR_FORCE") != "0"
}

func is256ColorSupported() bool {
	return isTrueColorSupported() ||
		strings.Contains(os.Getenv("TERM"), "256") ||
		strings.Contains(os.Getenv("COLORTERM"), "256")
}

func isTrueColorSupported() bool {
	term := os.Getenv("TERM")
	colorterm := os.Getenv("COLORTERM")

	return strings.Contains(term, "24bit") ||
		strings.Contains(term, "truecolor") ||
		strings.Contains(colorterm, "24bit") ||
		strings.Contains(colorterm, "truecolor")
}
