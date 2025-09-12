package message

import (
	"dubbo-admin-ai/plugins/model"
	"dubbo-admin-ai/schema"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// The standard output message structure of agent.
type Message struct {
	Schema schema.Schema       `json:"content,omitempty"`
	Model  *model.Model        `json:"model,omitempty"`
	Status schema.Status       `json:"status,omitempty" jsonschema:"enum=CONTINUED,enum=FINISHED,enum=PENDING,required=false"`
	Role   Role                `json:"role" jsonschema:"enum=user,enum=assistant,enum=system,enum=tool,required=true"`
	Usage  *ai.GenerationUsage `json:"usage,omitempty"`
}

func NewMessage(schema schema.Schema) *Message {
	return &Message{
		Role:   RoleUser,
		Schema: schema,
	}
}

func (m *Message) String() string {
	result := "Message[\n"
	result += fmt.Sprintf("  Role: %s\n", string(m.Role))

	if m.Status != "" {
		result += fmt.Sprintf("  Status: %s\n", string(m.Status))
	}

	if m.Model != nil {
		result += fmt.Sprintf("  Model: %s\n", m.Model.Key())
	}

	if m.Schema != nil {
		// Try to use String method if available, otherwise use JSON representation
		if stringer, ok := m.Schema.(fmt.Stringer); ok {
			result += fmt.Sprintf("  Schema: %s\n", stringer.String())
		} else {
			if schemaJSON, err := json.Marshal(m.Schema); err == nil {
				result += fmt.Sprintf("  Schema: %s\n", string(schemaJSON))
			} else {
				result += fmt.Sprintf("  Schema: %v\n", m.Schema)
			}
		}
	}

	if m.Usage != nil {
		if usageJSON, err := json.Marshal(m.Usage); err == nil {
			result += fmt.Sprintf("  Usage: %s\n", string(usageJSON))
		} else {
			result += fmt.Sprintf("  Usage: %v\n", m.Usage)
		}
	}

	result += "]"
	return result
}
