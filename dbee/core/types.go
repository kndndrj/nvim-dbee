package core

import "github.com/neovim/go-client/msgpack"

type SchemaType int

const (
	SchemaFul SchemaType = iota
	SchemaLess
)

type (
	// FormatterOpts provide various options for formatters
	FormatterOpts struct {
		SchemaType SchemaType
		ChunkStart int
	}

	// Formatter converts header and rows to bytes
	Formatter interface {
		Format(header Header, rows []Row, opts *FormatterOpts) ([]byte, error)
	}
)

type (
	// Row and Header are attributes of IterResult iterator
	Row    []any
	Header []string

	// Meta holds metadata
	Meta struct {
		// type of schema (schemaful or schemaless)
		SchemaType SchemaType
	}

	// ResultStream is a result from executed query and has a form of an iterator
	ResultStream interface {
		Meta() *Meta
		Header() Header
		Next() (Row, error)
		HasNext() bool
		Close()
	}
)

type StructureType int

const (
	StructureTypeNone StructureType = iota
	StructureTypeTable
	StructureTypeView
)

func (s StructureType) String() string {
	switch s {
	case StructureTypeNone:
		return ""
	case StructureTypeTable:
		return "table"
	case StructureTypeView:
		return "view"
	default:
		return ""
	}
}

// Structure represents the structure of a single database
type Structure struct {
	// Name to be displayed
	Name   string
	Schema string
	// Type of layout
	Type StructureType
	// Children layout nodes
	Children []Structure
}

func (l *Structure) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(&struct {
		Name     string      `msgpack:"name"`
		Schema   string      `msgpack:"schema"`
		Type     string      `msgpack:"type"`
		Children []Structure `msgpack:"children"`
	}{
		Name:     l.Name,
		Schema:   l.Schema,
		Type:     l.Type.String(),
		Children: l.Children,
	})
}
