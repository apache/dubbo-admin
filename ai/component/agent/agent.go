package agent

import (
	"context"
	"sync"
	"sync/atomic"

	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"
)

type Agent interface {
	Interact(context.Context, *schema.UserInput, string) *Channels
	GetMemory() *memory.HistoryMemory
}

// Channels carries streaming output back to the caller for one interaction.
// UserRespChan streams user-facing text/progress and the final answer;
// ErrorChan surfaces failures. A fresh Channels is created per interaction.
type Channels struct {
	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}
	nextIndex int
	traceID   string

	UserRespChan chan *schema.StreamFeedback
	ErrorChan    chan error
}

func (chans *Channels) SetTraceID(traceID string) { chans.traceID = traceID }

func (chans *Channels) TraceID() string { return chans.traceID }

func NewChannels(bufferSize int) *Channels {
	return &Channels{
		done:         make(chan struct{}),
		UserRespChan: make(chan *schema.StreamFeedback, bufferSize),
		ErrorChan:    make(chan error, bufferSize),
	}
}

// Close marks the Channels as finished without tearing down the underlying
// channels, so the consumer can drain any buffered messages.
func (chans *Channels) Close() {
	chans.closeOnce.Do(func() {
		chans.closed.Store(true)
		close(chans.done)
	})
}

func (chans *Channels) Closed() bool {
	return chans.closed.Load()
}

func (chans *Channels) Done() <-chan struct{} { return chans.done }

// Send assigns the next content-block index to the feedback and forwards it to
// the consumer. Sends for one interaction run sequentially (the strategy drives
// its steps on a single goroutine), so the counter needs no locking and never
// bleeds across sessions.
func (chans *Channels) Send(sf *schema.StreamFeedback) {
	sf.SetIndex(chans.nextIndex)
	chans.nextIndex++
	chans.UserRespChan <- sf
}

// EmitProgress streams a pre-rendered progress line to the consumer. A nil
// Channels is a no-op, so steps can run without a consumer (e.g. tests). The
// wording is owned by each strategy; this only handles delivery.
func EmitProgress(chans *Channels, text string) {
	if chans == nil {
		return
	}
	chans.Send(schema.NewStreamFeedback(text))
}
