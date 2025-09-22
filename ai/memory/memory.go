package memory

import (
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
)

const (
	SystemWindowLimit = 10
	CoreWindowLimit   = 20
	ChatWindowLimit   = 50

	// Block size limits
	MaxBlockSize      = 2048 // Maximum Block character count
	DefaultImportance = 5    // Default importance level
	MaxImportance     = 10   // Maximum importance level
)

type Memory struct {
	SystemWindow *MemoryWindow
	CoreWindow   *MemoryWindow
	ChatWindow   *MemoryWindow
}

type MemoryWindow struct {
	Blocks           []MemoryBlock
	EvictionStrategy EvictionStrategy
	WindowSize       int
}

type MemoryBlock interface {
	CompileToPrompt() *ai.Prompt
	Limit() int
	Size() int
	Priority() int
	UpdatePriority() error
	Reserved() bool
	Split(maxSize int) ([]MemoryBlock, error)
}

type EvictionStrategy interface {
	OnAccess(window *MemoryWindow) error
	OnInsert(window *MemoryWindow) error
	Evict(window *MemoryWindow) error
}

type FIFO struct {
}

type LRU struct {
}

// SystemMemoryBlock system memory block
type SystemMemoryBlock struct {
	content   string
	timestamp int64
	priority  int
	reserved  bool
}

func (s *SystemMemoryBlock) CompileToPrompt() *ai.Prompt {
	return nil
}

func (s *SystemMemoryBlock) Limit() int {
	return MaxBlockSize
}

func (s *SystemMemoryBlock) Size() int {
	return len(s.content)
}

func (s *SystemMemoryBlock) Priority() int {
	return s.priority
}

func (s *SystemMemoryBlock) UpdatePriority() error {
	s.timestamp = time.Now().UnixNano()
	s.priority = -int(s.timestamp / 1e6) // Convert to milliseconds and negate
	return nil
}

func (s *SystemMemoryBlock) Reserved() bool {
	return s.reserved
}

func (s *SystemMemoryBlock) Split(maxSize int) ([]MemoryBlock, error) {
	if s.Size() <= maxSize {
		return []MemoryBlock{s}, nil
	}

	parts := splitBySemanticBoundary(s.content, maxSize)
	blocks := make([]MemoryBlock, len(parts))

	for i, part := range parts {
		blocks[i] = &SystemMemoryBlock{
			content:   part,
			timestamp: s.timestamp,
			priority:  s.priority,
			reserved:  s.reserved,
		}
	}

	return blocks, nil
}

// CoreMemoryBlock core memory block
type CoreMemoryBlock struct {
	content    string
	timestamp  int64
	priority   int
	importance int
	reserved   bool
}

func (c *CoreMemoryBlock) CompileToPrompt() *ai.Prompt {
	// TODO: Need genkit registry to create prompt, return nil for now
	return nil
}

func (c *CoreMemoryBlock) Limit() int {
	return MaxBlockSize
}

func (c *CoreMemoryBlock) Size() int {
	return len(c.content)
}

func (c *CoreMemoryBlock) Priority() int {
	return c.priority
}

func (c *CoreMemoryBlock) UpdatePriority() error {
	c.timestamp = time.Now().UnixMilli()
	c.priority = calculatePriority(c.timestamp, c.importance)
	return nil
}

func (c *CoreMemoryBlock) Reserved() bool {
	return c.reserved
}

func (c *CoreMemoryBlock) Split(maxSize int) ([]MemoryBlock, error) {
	if c.Size() <= maxSize {
		return []MemoryBlock{c}, nil
	}

	parts := splitBySemanticBoundary(c.content, maxSize)
	blocks := make([]MemoryBlock, len(parts))

	for i, part := range parts {
		blocks[i] = &CoreMemoryBlock{
			content:    part,
			timestamp:  c.timestamp,
			priority:   c.priority,
			importance: c.importance,
			reserved:   c.reserved,
		}
	}

	return blocks, nil
}

// ChatHistoryMemoryBlock chat history memory block
type ChatHistoryMemoryBlock struct {
	content   string
	timestamp int64
	priority  int
	reserved  bool
}

func (ch *ChatHistoryMemoryBlock) CompileToPrompt() *ai.Prompt {
	// TODO: Need genkit registry to create prompt, return nil for now
	return nil
}

func (ch *ChatHistoryMemoryBlock) Limit() int {
	return MaxBlockSize
}

func (ch *ChatHistoryMemoryBlock) Size() int {
	return len(ch.content)
}

func (ch *ChatHistoryMemoryBlock) Priority() int {
	return ch.priority
}

func (ch *ChatHistoryMemoryBlock) UpdatePriority() error {
	ch.timestamp = time.Now().UnixNano()
	ch.priority = -int(ch.timestamp / 1e6) // Convert to milliseconds and negate
	return nil
}

func (ch *ChatHistoryMemoryBlock) Reserved() bool {
	return ch.reserved
}

func (ch *ChatHistoryMemoryBlock) Split(maxSize int) ([]MemoryBlock, error) {
	if ch.Size() <= maxSize {
		return []MemoryBlock{ch}, nil
	}

	parts := splitBySemanticBoundary(ch.content, maxSize)
	blocks := make([]MemoryBlock, len(parts))

	for i, part := range parts {
		blocks[i] = &ChatHistoryMemoryBlock{
			content:   part,
			timestamp: ch.timestamp,
			priority:  ch.priority,
			reserved:  ch.reserved,
		}
	}

	return blocks, nil
}

// FIFO strategy implementation
func (f *FIFO) OnAccess(window *MemoryWindow) error {
	// FIFO does not need to update priority on access
	return nil
}

func (f *FIFO) OnInsert(window *MemoryWindow) error {
	if len(window.Blocks) > 0 {
		lastBlock := window.Blocks[len(window.Blocks)-1]
		err := lastBlock.UpdatePriority()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FIFO) Evict(window *MemoryWindow) error {
	if len(window.Blocks) == 0 {
		return nil
	}

	minPriority := int(^uint(0) >> 1) // Max int value
	evictIndex := -1

	for i, block := range window.Blocks {
		if !block.Reserved() && block.Priority() < minPriority {
			minPriority = block.Priority()
			evictIndex = i
		}
	}

	if evictIndex >= 0 {
		// Remove found block
		window.Blocks = append(window.Blocks[:evictIndex], window.Blocks[evictIndex+1:]...)
	}

	return nil
}

// LRU strategy implementation
func (l *LRU) OnAccess(window *MemoryWindow) error {
	// Update priority of recently accessed block
	if len(window.Blocks) > 0 {
		lastBlock := window.Blocks[len(window.Blocks)-1]
		err := lastBlock.UpdatePriority()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *LRU) OnInsert(window *MemoryWindow) error {
	if len(window.Blocks) > 0 {
		lastBlock := window.Blocks[len(window.Blocks)-1]
		err := lastBlock.UpdatePriority()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *LRU) Evict(window *MemoryWindow) error {
	if len(window.Blocks) == 0 {
		return nil
	}

	minPriority := int(^uint(0) >> 1) // Max int value
	evictIndex := -1

	for i, block := range window.Blocks {
		if !block.Reserved() && block.Priority() < minPriority {
			minPriority = block.Priority()
			evictIndex = i
		}
	}

	if evictIndex >= 0 {
		// Remove found block
		window.Blocks = append(window.Blocks[:evictIndex], window.Blocks[evictIndex+1:]...)
	}

	return nil
}

// MemoryWindow method implementation
func (mw *MemoryWindow) Insert(block MemoryBlock) error {
	// Check if eviction is needed
	for mw.NeedsEviction() {
		err := mw.EvictionStrategy.Evict(mw)
		if err != nil {
			return err
		}
	}

	// Insert new block
	mw.Blocks = append(mw.Blocks, block)

	// Trigger insert event
	return mw.EvictionStrategy.OnInsert(mw)
}

func (mw *MemoryWindow) NeedsEviction() bool {
	return len(mw.Blocks) >= mw.WindowSize
}

func (mw *MemoryWindow) FindBlock(predicate func(MemoryBlock) bool) *MemoryBlock {
	for i := range mw.Blocks {
		if predicate(mw.Blocks[i]) {
			// Trigger access event
			mw.EvictionStrategy.OnAccess(mw)
			return &mw.Blocks[i]
		}
	}
	return nil
}

// Memory method implementation
func NewMemory() *Memory {
	return &Memory{
		SystemWindow: &MemoryWindow{
			Blocks:           make([]MemoryBlock, 0),
			EvictionStrategy: &FIFO{},
			WindowSize:       SystemWindowLimit,
		},
		CoreWindow: &MemoryWindow{
			Blocks:           make([]MemoryBlock, 0),
			EvictionStrategy: &LRU{},
			WindowSize:       CoreWindowLimit,
		},
		ChatWindow: &MemoryWindow{
			Blocks:           make([]MemoryBlock, 0),
			EvictionStrategy: &LRU{},
			WindowSize:       ChatWindowLimit,
		},
	}
}

func (m *Memory) AddSystemMemory(content string, reserved bool) error {
	block := &SystemMemoryBlock{
		content:   content,
		timestamp: time.Now().UnixNano(),
		priority:  0,
		reserved:  reserved,
	}
	block.UpdatePriority()

	// Check if splitting is needed
	if block.Size() > MaxBlockSize {
		blocks, err := block.Split(MaxBlockSize)
		if err != nil {
			return err
		}
		for _, b := range blocks {
			err := m.SystemWindow.Insert(b)
			if err != nil {
				return err
			}
		}
	} else {
		return m.SystemWindow.Insert(block)
	}

	return nil
}

func (m *Memory) AddCoreMemory(content string, importance int, reserved bool) error {
	block := &CoreMemoryBlock{
		content:    content,
		timestamp:  time.Now().UnixNano(),
		priority:   0,
		importance: importance,
		reserved:   reserved,
	}
	block.UpdatePriority()

	// Check if splitting is needed
	if block.Size() > MaxBlockSize {
		blocks, err := block.Split(MaxBlockSize)
		if err != nil {
			return err
		}
		for _, b := range blocks {
			err := m.CoreWindow.Insert(b)
			if err != nil {
				return err
			}
		}
	} else {
		return m.CoreWindow.Insert(block)
	}

	return nil
}

func (m *Memory) AddChatHistory(content string) error {
	block := &ChatHistoryMemoryBlock{
		content:   content,
		timestamp: time.Now().UnixNano(),
		priority:  0,
		reserved:  false,
	}
	block.UpdatePriority()

	// Check if splitting is needed
	if block.Size() > MaxBlockSize {
		blocks, err := block.Split(MaxBlockSize)
		if err != nil {
			return err
		}
		for _, b := range blocks {
			err := m.ChatWindow.Insert(b)
			if err != nil {
				return err
			}
		}
	} else {
		return m.ChatWindow.Insert(block)
	}

	return nil
}

func (m *Memory) CompileAllToPrompt() *ai.Prompt {
	// TODO: Need genkit registry to create unified prompt, return nil for now
	return nil
}

// Helper function implementation
func splitBySemanticBoundary(content string, maxSize int) []string {
	if len(content) <= maxSize {
		return []string{content}
	}

	var parts []string
	sentences := strings.Split(content, ".")
	currentPart := ""

	for _, sentence := range sentences {
		testPart := currentPart + sentence + "."
		if len(testPart) > maxSize && currentPart != "" {
			parts = append(parts, strings.TrimSpace(currentPart))
			currentPart = sentence + "."
		} else {
			currentPart = testPart
		}
	}

	if currentPart != "" {
		parts = append(parts, strings.TrimSpace(currentPart))
	}

	return parts
}

func calculatePriority(timestamp int64, importance int) int {
	timeScore := -int(timestamp / 1e6)      // Higher timestamp means higher priority
	importanceBonus := importance * 1000000 // Importance bonus
	return timeScore + importanceBonus
}
