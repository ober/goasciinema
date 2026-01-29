package sanitize

import (
	"regexp"
	"strings"
)

// ansiEscape matches ANSI escape sequences and terminal control characters.
// Mirrors the Python clean_logs.py ANSI_ESCAPE pattern.
var ansiEscape = regexp.MustCompile(
	`\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]` + // CSI sequences
		`|\x1b\].*?(?:\x07|\x1b\\)` + // OSC sequences (BEL or ST terminated)
		`|\x1b[PX^_][^\x1b]*\x1b\\` + // DCS, SOS, PM, APC sequences
		`|\x1b[()].` + // Charset designation
		`|\x1b[\x20-\x2f][\x30-\x7e]` + // nF escape sequences
		`|\x1b[\x40-\x5f]` + // Other Fe sequences (2-byte)
		`|\x1b.` + // Any remaining ESC + byte
		`|\x07` + // BEL
		`|\x08` + // Backspace
		`|\x0f` + // SI (Shift In)
		`|\x0e` + // SO (Shift Out)
		`|\r`, // Carriage return
)

// terminalArtifacts matches terminal control fragments that may remain
// after ESC-prefix stripping. Mirrors clean_logs.py TERMINAL_ARTIFACTS.
var terminalArtifacts = regexp.MustCompile(
	`\[\?[\d;]*[hlsr]` + // DEC private mode set/reset
		`|\[[\d;]*[HfABCDEFGJKLMPXZrd@` + "`" + `a]` + // Cursor control/erase
		`|\[[0-9;]*m`, // SGR (color) sequences
)

// multiSpaces matches runs of two or more spaces.
var multiSpaces = regexp.MustCompile(`  +`)

// StripANSI removes ANSI escape codes, terminal control characters, and
// artifacts from text. Matches the behavior of clean_logs.py clean_line().
func StripANSI(text string) string {
	// First pass: ANSI escapes and control characters
	text = ansiEscape.ReplaceAllString(text, "")
	// Second pass: terminal artifact fragments
	text = terminalArtifacts.ReplaceAllString(text, "")
	// Collapse multiple spaces to double space
	text = multiSpaces.ReplaceAllString(text, "  ")
	return text
}

// CleanLines applies StripANSI per line, trims trailing whitespace, and
// returns only non-empty lines joined by newlines.
func CleanLines(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		line = StripANSI(line)
		line = strings.TrimRight(line, " \t")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
