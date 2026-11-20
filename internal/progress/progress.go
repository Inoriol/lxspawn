package progress

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

type EventType string

const (
	EventDownloading  EventType = "downloading"
	EventExtracting   EventType = "extracting"
	EventProvisioning EventType = "provisioning"
	EventProbe        EventType = "probe"
	EventStarting     EventType = "starting"
	EventDone         EventType = "done"
	EventError        EventType = "error"
)

type Event struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message,omitempty"`
	Phase     string    `json:"phase,omitempty"`
	Progress  int       `json:"progress,omitempty"`
	Timestamp string    `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

type Reporter struct {
	mu       sync.Mutex
	writer   io.Writer
	jsonMode bool
}

func NewReporter(writer io.Writer, jsonMode bool) *Reporter {
	return &Reporter{
		writer:   writer,
		jsonMode: jsonMode,
	}
}

func (r *Reporter) Emit(eventType EventType, message string, phase string, progress int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := Event{
		Type:      eventType,
		Message:   message,
		Phase:     phase,
		Progress:  progress,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if r.jsonMode {
		data, _ := json.Marshal(event)
		fmt.Fprintln(r.writer, string(data))
	}
}

func (r *Reporter) EmitError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := Event{
		Type:      EventError,
		Error:     err.Error(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if r.jsonMode {
		data, _ := json.Marshal(event)
		fmt.Fprintln(r.writer, string(data))
	}
}

func (r *Reporter) EmitDone(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := Event{
		Type:      EventDone,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if r.jsonMode {
		data, _ := json.Marshal(event)
		fmt.Fprintln(r.writer, string(data))
	}
}
