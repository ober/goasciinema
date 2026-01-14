package asciicast

import (
	"time"
)

// Version constants
const (
	Version2 = 2
)

// Event types
const (
	EventTypeOutput = "o" // stdout output
	EventTypeInput  = "i" // stdin input
	EventTypeMarker = "m" // marker
	EventTypeResize = "r" // resize
)

// Header represents the asciicast v2 header
type Header struct {
	Version       int               `json:"version"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Timestamp     int64             `json:"timestamp,omitempty"`
	Duration      float64           `json:"duration,omitempty"`
	IdleTimeLimit float64           `json:"idle_time_limit,omitempty"`
	Command       string            `json:"command,omitempty"`
	Title         string            `json:"title,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Theme         *Theme            `json:"theme,omitempty"`
}

// Theme represents terminal color theme
type Theme struct {
	Foreground string `json:"fg,omitempty"`
	Background string `json:"bg,omitempty"`
	Palette    string `json:"palette,omitempty"`
}

// Event represents a single asciicast event
type Event struct {
	Time float64
	Type string
	Data string
}

// Recording represents a complete asciicast recording
type Recording struct {
	Header Header
	Events []Event
}

// NewHeader creates a new header with default values
func NewHeader(width, height int) Header {
	return Header{
		Version:   Version2,
		Width:     width,
		Height:    height,
		Timestamp: time.Now().Unix(),
		Env:       make(map[string]string),
	}
}
