package schema

import (
	"embed"
	"encoding/json"
)

//go:embed schemas/*
var embedded embed.FS

// StructuredOutputSchema mirrors the structure used by the OpenAI provider so
// we can reuse the same JSON schema files. Only the `Schema` field is used by
// the Gemini client – the wrapper itself is kept for parity and potential
// future needs.
type StructuredOutputSchema struct {
	Schema JSONSchema `json:"schema,omitempty"`
	Name   string     `json:"name,omitempty"`
	Strict bool       `json:"strict,omitempty"`
}

type JSONSchema struct {
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	//Required             []string               `json:"required,omitempty"`
	//AdditionalProperties bool                   `json:"additionalProperties"`
}

// GetWorkspaceChangeProposalSchema parses and returns the schema definition
// for workspace change proposals.
func GetWorkspaceChangeProposalSchema() JSONSchema {
	return getSchema("schemas/workspace_change_proposal_schema.json")
}

// GetModuleContextSchema returns the schema definition for module context
// generation.
func GetModuleContextSchema() JSONSchema {
	return getSchema("schemas/module_selfcontained_context_schema.json")
}

// GetModuleExternalContextSchema returns the schema definition used when
// requesting external contexts in bulk.
func GetModuleExternalContextSchema() JSONSchema {
	return getSchema("schemas/module_external_context_schema.json")
}

func getSchema(path string) JSONSchema {
	data, _ := embedded.ReadFile(path)
	var s JSONSchema
	_ = json.Unmarshal(data, &s) // the embedded asset is trusted
	return s
}
