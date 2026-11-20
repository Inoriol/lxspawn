package progress

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReporterEmit(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, true)

	r.Emit(EventProvisioning, "Creating machine", "create", 50)

	output := buf.String()
	if output == "" {
		t.Fatal("expected output, got empty string")
	}

	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event.Type != EventProvisioning {
		t.Errorf("expected type %s, got %s", EventProvisioning, event.Type)
	}
	if event.Message != "Creating machine" {
		t.Errorf("expected message 'Creating machine', got %q", event.Message)
	}
	if event.Phase != "create" {
		t.Errorf("expected phase 'create', got %q", event.Phase)
	}
	if event.Progress != 50 {
		t.Errorf("expected progress 50, got %d", event.Progress)
	}
	if event.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestReporterEmitError(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, true)

	r.EmitError(&testError{"something failed"})

	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event.Type != EventError {
		t.Errorf("expected type %s, got %s", EventError, event.Type)
	}
	if event.Error != "something failed" {
		t.Errorf("expected error 'something failed', got %q", event.Error)
	}
}

func TestReporterEmitDone(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, true)

	r.EmitDone("Machine created successfully")

	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event.Type != EventDone {
		t.Errorf("expected type %s, got %s", EventDone, event.Type)
	}
	if event.Message != "Machine created successfully" {
		t.Errorf("expected message 'Machine created successfully', got %q", event.Message)
	}
}

func TestReporterNonJSONMode(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, false)

	r.Emit(EventProvisioning, "test", "phase", 0)

	if buf.Len() != 0 {
		t.Errorf("expected no output in non-JSON mode, got %q", buf.String())
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
