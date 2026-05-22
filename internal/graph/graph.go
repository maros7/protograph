// Package graph defines the data model for protograph.
package graph

// ProtoGraph is the top-level structure containing all parsed proto information.
type ProtoGraph struct {
	Root     string        `json:"root"`
	Files    []ProtoFile   `json:"files"`
	Messages []Message     `json:"messages"`
	Enums    []Enum        `json:"enums"`
	Services []Service     `json:"services"`
	Edges    []DependsEdge `json:"edges"` // message A uses message B
}

// ProtoFile represents a parsed .proto file.
type ProtoFile struct {
	Path    string   `json:"path"`
	Package string   `json:"package"`
	Imports []string `json:"imports,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// Message represents a protobuf message.
type Message struct {
	Name       string   `json:"name"`
	FullName   string   `json:"full_name"` // package.MessageName
	File       string   `json:"file"`
	Line       int      `json:"line,omitempty"`
	Doc        string   `json:"doc,omitempty"`
	Fields     []Field  `json:"fields,omitempty"`
	Oneofs     []Oneof  `json:"oneofs,omitempty"`
	Nested     []string `json:"nested,omitempty"`     // nested message names
	Parent     string   `json:"parent,omitempty"`     // parent message if nested
	IsMapEntry bool     `json:"is_map_entry,omitempty"`
}

// Field represents a message field.
type Field struct {
	Name     string `json:"name"`
	Number   int    `json:"number"`
	Type     string `json:"type"`      // scalar type or message reference
	Label    string `json:"label"`     // optional, repeated, required, map
	JSONName string `json:"json_name,omitempty"`
	Doc      string `json:"doc,omitempty"`
	Default  string `json:"default,omitempty"`
	Deprecated bool  `json:"deprecated,omitempty"`
	OneofName  string `json:"oneof,omitempty"`
}

// Oneof represents a oneof group.
type Oneof struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"` // field names in this oneof
}

// Enum represents a protobuf enum.
type Enum struct {
	Name     string      `json:"name"`
	FullName string      `json:"full_name"`
	File     string      `json:"file"`
	Line     int         `json:"line,omitempty"`
	Doc      string      `json:"doc,omitempty"`
	Values   []EnumValue `json:"values"`
	Parent   string      `json:"parent,omitempty"` // parent message if nested
}

// EnumValue represents a single enum value.
type EnumValue struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
	Doc    string `json:"doc,omitempty"`
}

// Service represents a protobuf service (gRPC).
type Service struct {
	Name    string   `json:"name"`
	File    string   `json:"file"`
	Doc     string   `json:"doc,omitempty"`
	Methods []Method `json:"methods"`
}

// Method represents an RPC method in a service.
type Method struct {
	Name           string `json:"name"`
	InputType      string `json:"input_type"`
	OutputType     string `json:"output_type"`
	ClientStream   bool   `json:"client_stream,omitempty"`
	ServerStream   bool   `json:"server_stream,omitempty"`
	Doc            string `json:"doc,omitempty"`
}

// DependsEdge records that message A has a field of type B.
type DependsEdge struct {
	From      string `json:"from"`       // message full name
	To        string `json:"to"`         // referenced type full name
	FieldName string `json:"field_name"` // which field creates the dependency
}
