package models

type (
	// Row and Header are attributes of IterResult iterator
	Row    []any
	Header []string
)

type SchemaType int

const (
	SchemaFul SchemaType = iota
	SchemaLess
)

type (
	// FormatOpts provide various options for formatters
	FormatOpts struct {
		SchemaType SchemaType
		ChunkStart int
	}

	// Meta holds metadata
	Meta struct {
		// type of schema (schemaful or schemaless)
		SchemaType SchemaType
	}

	// IterResult is an iterator which provides rows and headers from the Input
	IterResult interface {
		Meta() *Meta
		Header() Header
		Next() (Row, error)
		HasNext() bool
		Close()
	}
)
