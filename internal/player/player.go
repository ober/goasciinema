package player

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ober/goasciinema/internal/asciicast"
	ttypkg "github.com/ober/goasciinema/internal/tty"
)

// Options configures the player
type Options struct {
	Speed         float64
	IdleTimeLimit float64
	Loop          bool
	MaxWait       float64
}

// Player handles asciicast playback
type Player struct {
	options Options
	paused  bool
	step    bool
}

// New creates a new player
func New(options Options) *Player {
	if options.Speed <= 0 {
		options.Speed = 1.0
	}
	return &Player{
		options: options,
	}
}

// Play plays the asciicast file
func (p *Player) Play(filename string) error {
	reader, err := asciicast.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	// Set terminal size if possible
	if ttypkg.IsTerminal(ttypkg.GetStdoutFd()) {
		fmt.Printf("\x1b[8;%d;%dt", reader.Header.Height, reader.Header.Width)
	}

	for {
		err := p.playOnce(reader)
		if err != nil {
			return err
		}

		if !p.options.Loop {
			break
		}

		// Reset reader for loop
		reader.Close()
		reader, err = asciicast.Open(filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Player) playOnce(reader *asciicast.Reader) error {
	var prevTime float64

	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Calculate delay
		delay := event.Time - prevTime
		prevTime = event.Time

		// Apply idle time limit
		if p.options.IdleTimeLimit > 0 && delay > p.options.IdleTimeLimit {
			delay = p.options.IdleTimeLimit
		}
		if p.options.MaxWait > 0 && delay > p.options.MaxWait {
			delay = p.options.MaxWait
		}

		// Apply speed
		delay = delay / p.options.Speed

		// Wait
		if delay > 0 {
			time.Sleep(time.Duration(delay * float64(time.Second)))
		}

		// Output only stdout events
		if event.Type == asciicast.EventTypeOutput {
			os.Stdout.WriteString(event.Data)
		}
	}
}

// Cat outputs the full recording without timing
func Cat(filename string) error {
	reader, err := asciicast.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if event.Type == asciicast.EventTypeOutput {
			os.Stdout.WriteString(event.Data)
		}
	}
}
