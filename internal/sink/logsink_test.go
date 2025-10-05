package sink

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shortontech/gotrack/internal/event"
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
func setupLogSink(t *testing.T, logPath string) (*LogSink, func()) {
	t.Helper()
	oldPath := os.Getenv("LOG_PATH")
	os.Setenv("LOG_PATH", logPath)
	sink := NewLogSink()
	ctx := context.Background()
	if err := sink.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	cleanup := func() {
		sink.Close()
		os.Setenv("LOG_PATH", oldPath)
	}
	return sink, cleanup
}

func verifyEventInLog(t *testing.T, logPath string, wantID, wantType string) {
	t.Helper()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	var decoded event.Event
	if err := json.Unmarshal(content[:len(content)-1], &decoded); err != nil {
		t.Fatalf("log content is not valid JSON: %v", err)
	}
	if decoded.EventID != wantID {
		t.Errorf("event_id = %q, want %q", decoded.EventID, wantID)
	}
	if decoded.Type != wantType {
		t.Errorf("type = %q, want %q", decoded.Type, wantType)
	}
}

func TestLogSinkEnqueue(t *testing.T) {
	t.Run("writes event to file", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "events.log")
		sink, cleanup := setupLogSink(t, logPath)
		defer cleanup()
		evt := event.Event{EventID: "test-123", Type: "pageview", TS: time.Now().Format(time.RFC3339)}
		if err := sink.Enqueue(evt); err != nil {
			t.Fatalf("Enqueue() failed: %v", err)
		}
		sink.Close()
		verifyEventInLog(t, logPath, "test-123", "pageview")
	})

	t.Run("appends multiple events with newlines", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "events.log")
		sink, cleanup := setupLogSink(t, logPath)
		defer cleanup()
		for i := 1; i <= 3; i++ {
			evt := event.Event{EventID: "test-" + string(rune('0'+i)), Type: "click"}
			if err := sink.Enqueue(evt); err != nil {
				t.Fatalf("Enqueue() failed: %v", err)
			}
		}
		sink.Close()
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}
		if len(content) == 0 {
			t.Error("log file should not be empty")
		}
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
		sink, cleanup := setupLogSink(t, "stdout")
		defer cleanup()
		evt := event.Event{EventID: "stdout-test", Type: "test"}
		if err := sink.Enqueue(evt); err != nil {
			t.Errorf("Enqueue() to stdout failed: %v", err)
		}
	})

	t.Run("handles concurrent writes safely", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "concurrent.log")
		sink, cleanup := setupLogSink(t, logPath)
		defer cleanup()
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				evt := event.Event{EventID: "concurrent-" + string(rune('0'+id)), Type: "test"}
				_ = sink.Enqueue(evt)
				done <- true
			}(i)
		}
		for i := 0; i < 10; i++ {
			<-done
		}
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
