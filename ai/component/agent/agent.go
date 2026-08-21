package agent

import (
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"
)

type Agent interface {
	Interact(*schema.UserInput, string) *Channels
	GetMemory() *memory.HistoryMemory
}

// Channels carries streaming output back to the caller for one interaction.
// UserRespChan streams user-facing text/progress and the final answer;
// ErrorChan surfaces failures. A fresh Channels is created per interaction.
type Channels struct {
	closed    bool
	nextIndex int

	UserRespChan chan *schema.StreamFeedback
	ErrorChan    chan error
}

func NewChannels(bufferSize int) *Channels {
	return &Channels{
		closed:       false,
		UserRespChan: make(chan *schema.StreamFeedback, bufferSize),
		ErrorChan:    make(chan error, bufferSize),
	}
}

// Close marks the Channels as finished without tearing down the underlying
// channels, so the consumer can drain any buffered messages.
func (chans *Channels) Close() {
	chans.closed = true
}

func (chans *Channels) Closed() bool {
	return chans.closed
}

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
