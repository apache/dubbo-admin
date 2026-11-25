package hooks

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
)

const (
	// usageAuditQueueSize bounds the number of metadata records waiting for a
	// Store write. Accounting is best effort: when the queue is full the event is
	// dropped instead of blocking the model/tool hot path.
	usageAuditQueueSize = 1024
	// usageAuditShutdownTimeout bounds the final drain in Component.Stop.
	usageAuditShutdownTimeout = 5 * time.Second
)

// usageAuditRecord carries one durable boundary's metadata plus the trusted
// identity needed to write it after the request has finished. It never carries
// prompt, response, or tool payloads.
type usageAuditRecord struct {
	access security.AccessContext
	event  Event
	audit  *conversationstore.AuditEvent
	usage  *conversationstore.UsageEvent
	// flushed is set by Flush; the worker closes it after all earlier records.
	flushed chan struct{}
}

func (r usageAuditRecord) persist(ctx context.Context, store conversationstore.UsageAuditStore, logger *slog.Logger) {
	if r.audit != nil {
		if err := store.RecordAudit(ctx, *r.audit); err != nil {
			logger.Warn("failed to record AI audit event", "event", r.event, "error_type", ErrorType(err))
		}
	}
	if r.usage != nil {
		if err := store.RecordUsage(ctx, *r.usage); err != nil {
			logger.Warn("failed to record AI usage event", "event", r.event, "error_type", ErrorType(err))
		}
	}
}

// usageAuditWriter decouples metadata persistence from the request path. A
// single worker preserves event order and keeps capacity-row contention to one
// writer per process instead of one per model/tool call.
type usageAuditWriter struct {
	store  conversationstore.UsageAuditStore
	logger *slog.Logger
	queue  chan usageAuditRecord
	stop   chan struct{}
	done   chan struct{}
	start  sync.Once
	close  sync.Once
	// dropped counts records discarded because the bounded queue was full.
	dropped atomic.Uint64
}

func newUsageAuditWriter(store conversationstore.UsageAuditStore, logger *slog.Logger, queueSize int) *usageAuditWriter {
	if logger == nil {
		logger = slog.Default()
	}
	if queueSize <= 0 {
		queueSize = usageAuditQueueSize
	}
	return &usageAuditWriter{
		store:  store,
		logger: logger,
		queue:  make(chan usageAuditRecord, queueSize),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (w *usageAuditWriter) Start() {
	if w == nil {
		return
	}
	w.start.Do(func() { go w.run() })
}

func (w *usageAuditWriter) Enqueue(record usageAuditRecord) {
	if w == nil {
		return
	}
	w.Start()
	select {
	case w.queue <- record:
	default:
		dropped := w.dropped.Add(1)
		// Avoid one log line per dropped event during a Store outage; the counter
		// still exposes the total for operators and tests.
		if dropped == 1 || dropped%100 == 0 {
			w.logger.Warn("usage audit queue full; dropping record", "event", record.event, "dropped_total", dropped)
		}
	}
}

// Flush blocks until every record enqueued before the call has been processed.
func (w *usageAuditWriter) Flush(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.Start()
	ack := make(chan struct{})
	select {
	case w.queue <- usageAuditRecord{flushed: ack}:
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-ack:
		return nil
	case <-w.done:
		// The worker drained or stopped before reaching the barrier.
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops accepting work, drains queued records within a bounded budget,
// and waits for the worker to exit.
func (w *usageAuditWriter) Close() {
	if w == nil {
		return
	}
	// A writer may be closed after a failed Init without ever handling an event;
	// start the worker so it can observe the stop signal and exit.
	w.Start()
	w.close.Do(func() { close(w.stop) })
	<-w.done
}

func (w *usageAuditWriter) Dropped() uint64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

func (w *usageAuditWriter) run() {
	defer close(w.done)
	for {
		// Prefer shutdown over a busy queue so Stop cannot be starved by a
		// constant stream of events.
		select {
		case <-w.stop:
			w.drain()
			return
		default:
		}
		select {
		case <-w.stop:
			w.drain()
			return
		case record := <-w.queue:
			w.handle(record, nil)
		}
	}
}

func (w *usageAuditWriter) drain() {
	ctx, cancel := context.WithTimeout(context.Background(), usageAuditShutdownTimeout)
	defer cancel()
	for {
		select {
		case record := <-w.queue:
			w.handle(record, ctx)
		default:
			return
		}
	}
}

func (w *usageAuditWriter) handle(record usageAuditRecord, drainCtx context.Context) {
	if record.flushed != nil {
		close(record.flushed)
		return
	}
	base := context.Background()
	if drainCtx != nil {
		base = drainCtx
	}
	ctx, cancel := context.WithTimeout(security.WithAccessContext(base, record.access), usageAuditPersistenceTimeout)
	defer cancel()
	record.persist(ctx, w.store, w.logger)
}
