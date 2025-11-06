package mapping

import (
	"fmt"
)

// DefinitionType represents the type of code definition
type DefinitionType int

const (
	DefFunction DefinitionType = iota
	DefMethod
	DefClass
	DefInterface
	DefStruct
	DefEnum
	DefType
	DefVariable
	DefConstant
	DefModule
	DefNamespace
)

// String returns the string representation of DefinitionType
func (d DefinitionType) String() string {
	return [...]string{
		"Function", "Method", "Class", "Interface", "Struct",
		"Enum", "Type", "Variable", "Constant", "Module", "Namespace",
	}[d]
}

// Visibility represents code visibility
type Visibility int

const (
	VisibilityPublic Visibility = iota
	VisibilityPrivate
	VisibilityProtected
	VisibilityInternal
)

// String returns the string representation of Visibility
func (v Visibility) String() string {
	return [...]string{
		"Public", "Private", "Protected", "Internal",
	}[v]
}

// Definition represents a code definition (function, class, method, etc.)
type Definition struct {
	Type          DefinitionType         `json:"type"`
	Name          string                 `json:"name"`
	QualifiedName string                 `json:"qualified_name"`
	FilePath      string                 `json:"file_path"`
	StartLine     int                    `json:"start_line"`
	EndLine       int                    `json:"end_line"`
	StartByte     int                    `json:"start_byte"`
	EndByte       int                    `json:"end_byte"`
	Signature     string                 `json:"signature"`
	DocComment    string                 `json:"doc_comment,omitempty"`
	Visibility    Visibility             `json:"visibility"`
	Parameters    []*Parameter           `json:"parameters,omitempty"`
	ReturnType    string                 `json:"return_type,omitempty"`
	Parent        string                 `json:"parent,omitempty"`
	Children      []string               `json:"children,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Parameter represents a function/method parameter
type Parameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
}

// Import represents an import statement
type Import struct {
	Path       string   `json:"path"`
	Alias      string   `json:"alias,omitempty"`
	Items      []string `json:"items,omitempty"`
	IsRelative bool     `json:"is_relative"`
	StartLine  int      `json:"start_line"`
}

// Export represents an export statement
type Export struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsDefault bool   `json:"is_default"`
	StartLine int    `json:"start_line"`
}

// Comment represents a code comment
type Comment struct {
	Text      string `json:"text"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	IsDoc     bool   `json:"is_doc"`
}

// Reference represents a reference to a definition
type Reference struct {
	DefinitionID string `json:"definition_id"`
	FilePath     string `json:"file_path"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Context      string `json:"context"`
}

// String returns a string representation of a Definition
func (d *Definition) String() string {
	return fmt.Sprintf("%s %s (%s:%d-%d)",
		d.Type, d.QualifiedName, d.FilePath, d.StartLine, d.EndLine)
}

// GetSignature returns a formatted signature for the definition
func (d *Definition) GetSignature() string {
	if d.Signature != "" {
		return d.Signature
	}

	// Generate signature based on type
	switch d.Type {
	case DefFunction, DefMethod:
		sig := d.Name + "("
		for i, param := range d.Parameters {
			if i > 0 {
				sig += ", "
			}
			sig += param.Name
			if param.Type != "" {
				sig += " " + param.Type
			}
		}
		sig += ")"
		if d.ReturnType != "" {
			sig += " " + d.ReturnType
		}
		return sig
	default:
		return d.Name
	}
}

// LineCount returns the number of lines in the definition
func (d *Definition) LineCount() int {
	return d.EndLine - d.StartLine + 1
}
