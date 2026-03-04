package utils

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
)

// ToGenkitDocument converts an Eino schema.Document to a Genkit ai.Document
func ToGenkitDocument(doc *schema.Document) *ai.Document {
	if doc == nil {
		return nil
	}
	// Create text part
	part := ai.NewTextPart(doc.Content)

	// Copy metadata
	meta := make(map[string]any)
	maps.Copy(meta, doc.MetaData)
	// Store ID in metadata if present
	if doc.ID != "" {
		meta["_id"] = doc.ID
	}

	return &ai.Document{
		Content:  []*ai.Part{part},
		Metadata: meta,
	}
}

// ToEinoDocument converts a Genkit ai.Document to an Eino schema.Document
func ToEinoDocument(doc *ai.Document) *schema.Document {
	if doc == nil {
		return nil
	}

	// Extract text content
	var contentBuilder strings.Builder
	for _, part := range doc.Content {
		if part.IsText() {
			contentBuilder.WriteString(part.Text)
		}
	}

	// Copy metadata and extract ID
	meta := make(map[string]any)
	var id string
	maps.Copy(meta, doc.Metadata)
	if strID, ok := doc.Metadata["_id"].(string); ok {
		id = strID
	}

	return &schema.Document{
		ID:       id,
		Content:  contentBuilder.String(),
		MetaData: meta,
	}
}

// ToGenkitDocuments converts a slice of Eino schema.Documents to Genkit ai.Documents
func ToGenkitDocuments(docs []*schema.Document) []*ai.Document {
	if docs == nil {
		return nil
	}
	res := make([]*ai.Document, len(docs))
	for i, doc := range docs {
		res[i] = ToGenkitDocument(doc)
	}
	return res
}

// ToEinoDocuments converts a slice of Genkit ai.Documents to Eino schema.Documents
func ToEinoDocuments(docs []*ai.Document) []*schema.Document {
	if docs == nil {
		return nil
	}
	res := make([]*schema.Document, len(docs))
	for i, doc := range docs {
		res[i] = ToEinoDocument(doc)
	}
	return res
}

// CopyFile copies source file content to target file, creates the file if target doesn't exist
// srcPath: source file path
// dstPath: target file path
func CopyFile(srcPath, dstPath string) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info %s: %w", srcPath, err)
	}

	// Ensure target directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", dstDir, err)
	}

	// Create or overwrite target file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// Copy file content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Set target file permissions same as source file
	if err := os.Chmod(dstPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

type Window[T any] struct {
	limit int
	begin int
	end   int
	data  []T
}

func NewWindow[T any](limit int) *Window[T] {
	if limit <= 0 {
		panic("limit must be positive")
	}
	return &Window[T]{
		limit: limit,
		data:  make([]T, limit+1),
		begin: 0,
		end:   0,
	}
}

func (w *Window[T]) Push(elm T) bool {
	if w.IsFull() {
		return false
	}
	w.data[w.end] = elm
	w.end++
	return true
}

func (w *Window[T]) IsEmpty() bool {
	return w.begin == w.end
}

func (w *Window[T]) IsFull() bool {
	return w.end == w.limit
}

func (w *Window[T]) Pop() T {
	if w.IsEmpty() {
		panic("window is empty")
	}
	val := w.data[w.begin]
	w.begin++
	return val
}

func (w *Window[T]) Size() int {
	return w.end - w.begin
}

func (w *Window[T]) Capacity() int {
	return w.limit
}

func (w *Window[T]) GetAll() []T {
	return w.data
}

func (w *Window[T]) GetCurData() T {
	if w.IsEmpty() {
		panic("window is empty")
	}
	return w.data[w.end-1]
}

func (w *Window[T]) GetWindow() []T {
	return w.data[w.begin:w.end]
}

func (w *Window[T]) GetWindowBounds() (begin, end int) {
	return w.begin, w.end
}
