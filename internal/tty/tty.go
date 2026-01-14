package tty

import (
	"os"

	"golang.org/x/term"
)

// RawMode puts the terminal in raw mode and returns a restore function
func RawMode(fd int) (func() error, error) {
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() error {
		return term.Restore(fd, oldState)
	}, nil
}

// GetSize returns the terminal dimensions
func GetSize(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

// IsTerminal returns whether the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// GetStdinFd returns stdin file descriptor
func GetStdinFd() int {
	return int(os.Stdin.Fd())
}

// GetStdoutFd returns stdout file descriptor
func GetStdoutFd() int {
	return int(os.Stdout.Fd())
}
