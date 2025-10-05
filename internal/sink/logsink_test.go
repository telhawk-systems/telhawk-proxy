package sink

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"revinar.io/go.track/internal/event"
)

// TestNewLogSink tests LogSink creation
func TestNewLogSink(t *testing.T) {
	t.Run("uses default path when env not set", func(t *testing.T) {
		// Clear any existing env var
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Unsetenv("LOG_PATH")

		sink := NewLogSink()
		if sink.dst != "ndjson.log" {
			t.Errorf("dst = %q, want ndjson.log", sink.dst)
		}
	})

	t.Run("uses env variable when set", func(t *testing.T) {
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)

		os.Setenv("LOG_PATH", "/tmp/custom.log")
		sink := NewLogSink()
		if sink.dst != "/tmp/custom.log" {
			t.Errorf("dst = %q, want /tmp/custom.log", sink.dst)
		}
	})
}

// TestLogSinkStart tests starting the log sink
func TestLogSinkStart(t *testing.T) {
	t.Run("creates file at destination path", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", logPath)

		sink := NewLogSink()
		ctx := context.Background()

		err := sink.Start(ctx)
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		defer sink.Close()

		// Verify file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("log file was not created at %s", logPath)
		}
	})

	t.Run("handles stdout mode", func(t *testing.T) {
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", "stdout")

		sink := NewLogSink()
		ctx := context.Background()

		err := sink.Start(ctx)
		if err != nil {
			t.Fatalf("Start() failed for stdout: %v", err)
		}

		// stdout mode should not set file pointer
		if sink.f != nil {
			t.Error("file pointer should be nil for stdout mode")
		}

		sink.Close()
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)

		// Try to create file in non-existent directory
		os.Setenv("LOG_PATH", "/nonexistent/directory/test.log")

		sink := NewLogSink()
		ctx := context.Background()

		err := sink.Start(ctx)
		if err == nil {
			t.Error("Start() should fail for invalid path")
			sink.Close()
		}
	})
}

// TestLogSinkEnqueue tests enqueueing events
func TestLogSinkEnqueue(t *testing.T) {
	t.Run("writes event to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "events.log")

		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", logPath)

		sink := NewLogSink()
		ctx := context.Background()

		if err := sink.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		defer sink.Close()

		// Enqueue an event
		evt := event.Event{
			EventID: "test-123",
			Type:    "pageview",
			TS:      time.Now().Format(time.RFC3339),
		}

		err := sink.Enqueue(evt)
		if err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}

		// Close to flush
		sink.Close()

		// Read the file and verify content
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		// Verify it's valid JSON
		var decoded event.Event
		if err := json.Unmarshal(content[:len(content)-1], &decoded); err != nil {
			t.Fatalf("log content is not valid JSON: %v", err)
		}

		if decoded.EventID != "test-123" {
			t.Errorf("event_id = %q, want test-123", decoded.EventID)
		}
		if decoded.Type != "pageview" {
			t.Errorf("type = %q, want pageview", decoded.Type)
		}
	})

	t.Run("appends multiple events with newlines", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "events.log")

		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", logPath)

		sink := NewLogSink()
		ctx := context.Background()

		if err := sink.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		defer sink.Close()

		// Enqueue multiple events
		for i := 1; i <= 3; i++ {
			evt := event.Event{
				EventID: "test-" + string(rune('0'+i)),
				Type:    "click",
			}
			if err := sink.Enqueue(evt); err != nil {
				t.Fatalf("Enqueue() failed: %v", err)
			}
		}

		sink.Close()

		// Read and verify
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		lines := len(content)
		if lines == 0 {
			t.Error("log file should not be empty")
		}

		// Count newlines
		newlineCount := 0
		for _, b := range content {
			if b == '\n' {
				newlineCount++
			}
		}

		if newlineCount != 3 {
			t.Errorf("expected 3 newlines, got %d", newlineCount)
		}
	})

	t.Run("handles stdout mode without error", func(t *testing.T) {
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", "stdout")

		sink := NewLogSink()
		ctx := context.Background()

		if err := sink.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		defer sink.Close()

		evt := event.Event{
			EventID: "stdout-test",
			Type:    "test",
		}

		// Should not error even though we can't easily verify stdout
		err := sink.Enqueue(evt)
		if err != nil {
			t.Errorf("Enqueue() to stdout failed: %v", err)
		}
	})

	t.Run("handles concurrent writes safely", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "concurrent.log")

		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", logPath)

		sink := NewLogSink()
		ctx := context.Background()

		if err := sink.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		defer sink.Close()

		// Write concurrently
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				evt := event.Event{
					EventID: "concurrent-" + string(rune('0'+id)),
					Type:    "test",
				}
				_ = sink.Enqueue(evt)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should not panic or corrupt the file
	})
}

// TestLogSinkClose tests closing the log sink
func TestLogSinkClose(t *testing.T) {
	t.Run("closes file handle", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "closeable.log")

		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", logPath)

		sink := NewLogSink()
		ctx := context.Background()

		if err := sink.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		err := sink.Close()
		if err != nil {
			t.Errorf("Close() failed: %v", err)
		}

		// File handle should be nil or closed
		// Try writing after close should not panic
		evt := event.Event{EventID: "after-close"}
		_ = sink.Enqueue(evt) // Should not panic
	})

	t.Run("handles close without start", func(t *testing.T) {
		sink := NewLogSink()
		err := sink.Close()
		if err != nil {
			t.Errorf("Close() on unstarted sink should not error: %v", err)
		}
	})

	t.Run("handles stdout mode close", func(t *testing.T) {
		oldPath := os.Getenv("LOG_PATH")
		defer os.Setenv("LOG_PATH", oldPath)
		os.Setenv("LOG_PATH", "stdout")

		sink := NewLogSink()
		ctx := context.Background()
		sink.Start(ctx)

		err := sink.Close()
		if err != nil {
			t.Errorf("Close() for stdout mode failed: %v", err)
		}
	})
}

// TestLogSinkName tests the Name method
func TestLogSinkName(t *testing.T) {
	sink := NewLogSink()
	if sink.Name() != "log" {
		t.Errorf("Name() = %q, want log", sink.Name())
	}
}

// TestLogSinkAppendMode tests that log sink appends to existing files
func TestLogSinkAppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "append.log")

	oldPath := os.Getenv("LOG_PATH")
	defer os.Setenv("LOG_PATH", oldPath)
	os.Setenv("LOG_PATH", logPath)

	// First write
	sink1 := NewLogSink()
	ctx := context.Background()
	if err := sink1.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	evt1 := event.Event{EventID: "first"}
	sink1.Enqueue(evt1)
	sink1.Close()

	// Second write (should append)
	sink2 := NewLogSink()
	if err := sink2.Start(ctx); err != nil {
		t.Fatalf("Second Start() failed: %v", err)
	}

	evt2 := event.Event{EventID: "second"}
	sink2.Enqueue(evt2)
	sink2.Close()

	// Read and verify both events exist
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "first") {
		t.Error("first event not found in log")
	}
	if !contains(contentStr, "second") {
		t.Error("second event not found in log")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
