package sink

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL driver
	"revinar.io/go.track/internal/event"
)

// PGConfig holds configuration for PostgreSQL sink
type PGConfig struct {
	DSN       string
	Table     string
	BatchSize int
	FlushMS   int
	UseCopy   bool
}

// PGSink implements high-throughput PostgreSQL ingestion with COPY support
type PGSink struct {
	config PGConfig
	db     *sql.DB
	
	// Batching
	batch      []event.Event
	batchMutex sync.Mutex
	flushTimer *time.Timer
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewPGSinkFromEnv creates a PGSink from environment variables
func NewPGSinkFromEnv() *PGSink {
	config := PGConfig{
		DSN:       getEnvOr("PG_DSN", "postgres://user:pass@localhost:5432/analytics?sslmode=disable"),
		Table:     getEnvOr("PG_TABLE", "events_json"),
		BatchSize: getIntEnv("PG_BATCH_SIZE", 500),
		FlushMS:   getIntEnv("PG_FLUSH_MS", 500),
		UseCopy:   getBoolEnv("PG_COPY", true),
	}
	
	return &PGSink{config: config}
}

// NewPGSink creates a PGSink with explicit configuration
func NewPGSink(dsn string) *PGSink {
	return &PGSink{
		config: PGConfig{
			DSN:       dsn,
			Table:     "events_json",
			BatchSize: 500,
			FlushMS:   500,
			UseCopy:   true,
		},
	}
}

func (s *PGSink) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})
	s.batch = make([]event.Event, 0, s.config.BatchSize)
	
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", s.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}
	
	// Test connection
	if err := db.PingContext(s.ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping postgres: %w", err)
	}
	
	s.db = db
	
	// Create table and indexes if they don't exist
	if err := s.ensureSchema(); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}
	
	// Start flush timer routine
	go s.flushRoutine()
	
	return nil
}

func (s *PGSink) Enqueue(e event.Event) error {
	s.batchMutex.Lock()
	defer s.batchMutex.Unlock()
	
	s.batch = append(s.batch, e)
	
	// If batch is full, flush immediately
	if len(s.batch) >= s.config.BatchSize {
		return s.flushBatch()
	}
	
	// Reset flush timer
	if s.flushTimer != nil {
		s.flushTimer.Stop()
	}
	s.flushTimer = time.AfterFunc(time.Duration(s.config.FlushMS)*time.Millisecond, func() {
		s.batchMutex.Lock()
		defer s.batchMutex.Unlock()
		s.flushBatch()
	})
	
	return nil
}

func (s *PGSink) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	
	// Wait for flush routine to finish
	if s.done != nil {
		<-s.done
	}
	
	// Flush any remaining events
	s.batchMutex.Lock()
	s.flushBatch()
	s.batchMutex.Unlock()
	
	if s.db != nil {
		return s.db.Close()
	}
	
	return nil
}

func (s *PGSink) Name() string {
	return "postgres"
}

// ensureSchema creates the table and indexes if they don't exist
func (s *PGSink) ensureSchema() error {
	// Create table
	createTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			event_id UUID UNIQUE NOT NULL,
			ts TIMESTAMPTZ NOT NULL DEFAULT now(),
			payload JSONB NOT NULL
		)`, s.config.Table)
	
	if _, err := s.db.ExecContext(s.ctx, createTable); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	
	// Create indexes
	indexes := []string{
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_ts ON %s (ts)", s.config.Table, s.config.Table),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_gin ON %s USING GIN (payload)", s.config.Table, s.config.Table),
	}
	
	for _, idx := range indexes {
		if _, err := s.db.ExecContext(s.ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	return nil
}

// flushRoutine handles periodic flushing and cleanup
func (s *PGSink) flushRoutine() {
	defer close(s.done)
	
	ticker := time.NewTicker(time.Duration(s.config.FlushMS) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.batchMutex.Lock()
			s.flushBatch()
			s.batchMutex.Unlock()
		}
	}
}

// flushBatch writes the current batch to PostgreSQL (must be called with mutex held)
func (s *PGSink) flushBatch() error {
	if len(s.batch) == 0 {
		return nil
	}
	
	var err error
	if s.config.UseCopy {
		err = s.flushWithCopy()
	} else {
		err = s.flushWithInsert()
	}
	
	if err != nil {
		// In production, you might want to handle this more gracefully
		// (e.g., retry, dead letter queue, etc.)
		fmt.Fprintf(os.Stderr, "PostgreSQL flush error: %v\n", err)
	} else {
		// Clear the batch on successful flush
		s.batch = s.batch[:0]
	}
	
	return err
}

// flushWithCopy uses COPY for high-throughput ingestion
func (s *PGSink) flushWithCopy() error {
	txn, err := s.db.BeginTx(s.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer txn.Rollback()
	
	// Prepare COPY statement
	stmt, err := txn.PrepareContext(s.ctx, pq.CopyIn(s.config.Table, "event_id", "ts", "payload"))
	if err != nil {
		return fmt.Errorf("failed to prepare copy: %w", err)
	}
	defer stmt.Close()
	
	// Add events to COPY
	for _, e := range s.batch {
		payload, err := json.Marshal(e)
		if err != nil {
			continue // Skip invalid events
		}
		
		var ts time.Time
		if e.TS != "" {
			if parsed, err := time.Parse(time.RFC3339, e.TS); err == nil {
				ts = parsed
			} else {
				ts = time.Now()
			}
		} else {
			ts = time.Now()
		}
		
		_, err = stmt.ExecContext(s.ctx, e.EventID, ts, string(payload))
		if err != nil {
			// Skip events with constraint violations (duplicate event_id)
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				continue // Unique violation - idempotent behavior
			}
			return fmt.Errorf("failed to exec copy: %w", err)
		}
	}
	
	// Execute the COPY
	if _, err = stmt.ExecContext(s.ctx); err != nil {
		return fmt.Errorf("failed to execute copy: %w", err)
	}
	
	return txn.Commit()
}

// flushWithInsert uses multi-value INSERT with ON CONFLICT for idempotency
func (s *PGSink) flushWithInsert() error {
	if len(s.batch) == 0 {
		return nil
	}
	
	// Build multi-value INSERT
	placeholders := make([]string, len(s.batch))
	args := make([]interface{}, len(s.batch)*3)
	
	for i, e := range s.batch {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		
		// event_id
		args[i*3] = e.EventID
		
		// timestamp
		var ts time.Time
		if e.TS != "" {
			if parsed, err := time.Parse(time.RFC3339, e.TS); err == nil {
				ts = parsed
			} else {
				ts = time.Now()
			}
		} else {
			ts = time.Now()
		}
		args[i*3+1] = ts
		
		// payload as JSONB
		payload, err := json.Marshal(e)
		if err != nil {
			payload = []byte("{}") // Fallback to empty object
		}
		args[i*3+2] = string(payload)
	}
	
	query := fmt.Sprintf(`
		INSERT INTO %s (event_id, ts, payload) 
		VALUES %s 
		ON CONFLICT (event_id) DO NOTHING`,
		s.config.Table,
		strings.Join(placeholders, ", "))
	
	_, err := s.db.ExecContext(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}
	
	return nil
}

// Helper functions
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
