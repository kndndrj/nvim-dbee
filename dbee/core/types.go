package core

import "strings"

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

func StructureTypeFromString(s string) StructureType {
	switch strings.ToLower(s) {
	case "table":
		return StructureTypeTable
	case "view":
		return StructureTypeView
	default:
		return StructureTypeNone
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
	Children []*Structure
}

type Columns struct {
	// Column name
	Name string `msgpack:"name"`
	// Data type msgpack
	Type string `msgpack:"type"`
}
