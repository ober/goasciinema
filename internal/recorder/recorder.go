package recorder

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/ober/goasciinema/internal/asciicast"
	ttypkg "github.com/ober/goasciinema/internal/tty"
)

// Options configures the recorder
type Options struct {
	Command       string
	Title         string
	IdleTimeLimit float64
	RecordStdin   bool
	Append        bool
	Cols          int
	Rows          int
	Env           []string
}

// Recorder handles terminal recording
type Recorder struct {
	options   Options
	writer    *asciicast.Writer
	startTime time.Time
	mu        sync.Mutex
}

// New creates a new recorder
func New(options Options) *Recorder {
	return &Recorder{
		options: options,
	}
}

// Record starts recording to the specified file
func (r *Recorder) Record(filename string) error {
	// Get terminal size
	cols, rows := r.options.Cols, r.options.Rows
	if cols == 0 || rows == 0 {
		var err error
		cols, rows, err = ttypkg.GetSize(ttypkg.GetStdoutFd())
		if err != nil {
			cols, rows = 80, 24 // Default size
		}
	}

	// Create header
	header := asciicast.NewHeader(cols, rows)
	header.Title = r.options.Title
	header.IdleTimeLimit = r.options.IdleTimeLimit
	header.Command = r.options.Command

	// Set environment
	header.Env = map[string]string{
		"SHELL": os.Getenv("SHELL"),
		"TERM":  os.Getenv("TERM"),
	}

	// Create writer
	writer, err := asciicast.NewWriter(filename, header, r.options.Append)
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}
	defer writer.Close()

	r.writer = writer

	// Determine shell/command to run
	shell := r.options.Command
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	// Create command
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "GOASCIINEMA_REC=1")

	// Start PTY
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Set up raw mode on stdin
	restore, err := ttypkg.RawMode(ttypkg.GetStdinFd())
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer restore()

	// Handle window size changes
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			if newCols, newRows, err := ttypkg.GetSize(ttypkg.GetStdoutFd()); err == nil {
				pty.Setsize(ptmx, &pty.Winsize{
					Rows: uint16(newRows),
					Cols: uint16(newCols),
				})
				r.writeResize(newCols, newRows)
			}
		}
	}()
	defer signal.Stop(sigCh)

	r.startTime = time.Now()

	// Copy stdin to pty
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				data := buf[:n]
				ptmx.Write(data)
				if r.options.RecordStdin {
					r.writeInput(string(data))
				}
			}
		}
	}()

	// Copy pty output to stdout and record
	buf := make([]byte, 32768)
	for {
		n, err := ptmx.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			// PTY closed
			break
		}
		if n > 0 {
			data := buf[:n]
			os.Stdout.Write(data)
			r.writeOutput(string(data))
		}
	}

	// Wait for command to finish
	cmd.Wait()

	return nil
}

func (r *Recorder) elapsedTime() float64 {
	return time.Since(r.startTime).Seconds()
}

func (r *Recorder) writeOutput(data string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer.WriteOutput(r.elapsedTime(), data)
}

func (r *Recorder) writeInput(data string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer.WriteInput(r.elapsedTime(), data)
}

func (r *Recorder) writeResize(cols, rows int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer.WriteResize(r.elapsedTime(), cols, rows)
}
