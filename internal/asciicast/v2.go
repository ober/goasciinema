package asciicast

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Writer writes asciicast v2 format
type Writer struct {
	file       *os.File
	mu         sync.Mutex
	timeOffset float64
}

// NewWriter creates a new asciicast v2 writer
func NewWriter(filename string, header Header, append bool) (*Writer, error) {
	var file *os.File
	var err error
	var timeOffset float64

	if append {
		// Check if file exists and read last timestamp
		if info, statErr := os.Stat(filename); statErr == nil && info.Size() > 0 {
			timeOffset, err = getLastTimestamp(filename)
			if err != nil {
				return nil, fmt.Errorf("failed to get last timestamp: %w", err)
			}
			file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open file for append: %w", err)
			}
			return &Writer{file: file, timeOffset: timeOffset}, nil
		}
	}

	file, err = os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	// Write header
	headerBytes, err := json.Marshal(header)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := file.Write(headerBytes); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	return &Writer{file: file, timeOffset: timeOffset}, nil
}

// WriteEvent writes a single event
func (w *Writer) WriteEvent(event Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Adjust timestamp with offset
	adjustedTime := event.Time + w.timeOffset

	// Format: [timestamp, "type", "data"]
	eventData := []interface{}{
		roundTimestamp(adjustedTime),
		event.Type,
		event.Data,
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := w.file.Write(eventBytes); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	if _, err := w.file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// WriteOutput writes an output event
func (w *Writer) WriteOutput(timestamp float64, data string) error {
	return w.WriteEvent(Event{Time: timestamp, Type: EventTypeOutput, Data: data})
}

// WriteInput writes an input event
func (w *Writer) WriteInput(timestamp float64, data string) error {
	return w.WriteEvent(Event{Time: timestamp, Type: EventTypeInput, Data: data})
}

// WriteMarker writes a marker event
func (w *Writer) WriteMarker(timestamp float64, label string) error {
	return w.WriteEvent(Event{Time: timestamp, Type: EventTypeMarker, Data: label})
}

// WriteResize writes a resize event
func (w *Writer) WriteResize(timestamp float64, cols, rows int) error {
	return w.WriteEvent(Event{Time: timestamp, Type: EventTypeResize, Data: fmt.Sprintf("%dx%d", cols, rows)})
}

// Close closes the writer
func (w *Writer) Close() error {
	return w.file.Close()
}

// Reader reads asciicast v2 format
type Reader struct {
	Header Header
	file   *os.File
	reader *bufio.Reader
}

// Open opens an asciicast file for reading
func Open(filename string) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	reader := bufio.NewReader(file)

	// Read header line
	headerLine, err := reader.ReadBytes('\n')
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerLine, &header); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	return &Reader{
		Header: header,
		file:   file,
		reader: reader,
	}, nil
}

// ReadEvent reads the next event
func (r *Reader) ReadEvent() (*Event, error) {
	line, err := r.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read event: %w", err)
	}

	// Skip empty lines
	if len(line) <= 1 {
		return r.ReadEvent()
	}

	var eventData []interface{}
	if err := json.Unmarshal(line, &eventData); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	if len(eventData) < 3 {
		return nil, fmt.Errorf("invalid event format")
	}

	timestamp, ok := eventData[0].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp type")
	}

	eventType, ok := eventData[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid event type")
	}

	data, ok := eventData[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid event data type")
	}

	return &Event{
		Time: timestamp,
		Type: eventType,
		Data: data,
	}, nil
}

// Events returns a channel of events
func (r *Reader) Events() <-chan Event {
	ch := make(chan Event)
	go func() {
		defer close(ch)
		for {
			event, err := r.ReadEvent()
			if err != nil {
				return
			}
			ch <- *event
		}
	}()
	return ch
}

// Close closes the reader
func (r *Reader) Close() error {
	return r.file.Close()
}

// Helper functions

func roundTimestamp(t float64) float64 {
	return float64(int64(t*1000000)) / 1000000
}

func getLastTimestamp(filename string) (float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var lastTimestamp float64

	// Skip header
	_, err = reader.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		if len(line) <= 1 {
			continue
		}

		var eventData []interface{}
		if err := json.Unmarshal(line, &eventData); err != nil {
			continue
		}

		if len(eventData) >= 1 {
			if ts, ok := eventData[0].(float64); ok {
				lastTimestamp = ts
			}
		}
	}

	return lastTimestamp, nil
}
