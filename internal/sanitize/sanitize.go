package sanitize

import (
	"regexp"
	"strings"
)

// ansiEscape matches ANSI escape sequences
var ansiEscape = regexp.MustCompile(
	`\x1b\[[0-9;]*[a-zA-Z]` + // CSI sequences (colors, cursor, etc.)
		`|\x1b\][^\x07]*\x07` + // OSC sequences (title, etc.)
		`|\x1b[()][AB012]` + // Character set selection
		`|\x1b\[\?[0-9;]*[a-zA-Z]` + // Private mode sequences
		`|\x1b[PX^_][^\x1b]*\x1b\\` + // DCS, SOS, PM, APC sequences
		`|\x1b.`, // Other two-byte sequences
)

// StripANSI removes ANSI escape codes and non-printable characters from text.
// Keeps only printable ASCII (0x20-0x7E) plus newline, tab, and carriage return.
func StripANSI(text string) string {
	// Remove ANSI escape sequences
	text = ansiEscape.ReplaceAllString(text, "")

	// Keep only printable ASCII plus newline, tab, carriage return
	var result strings.Builder
	result.Grow(len(text))

	for _, char := range text {
		if (char >= 0x20 && char <= 0x7E) || char == '\n' || char == '\t' || char == '\r' {
			result.WriteRune(char)
		}
	}

	return result.String()
}
