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
	// Meta holds metadata
	Meta struct {
		// type of schema (shcemaful or schemaless)
		SchemaType SchemaType
		// position of the first row of the result - if the result is from row 500 to 1000, this nubmer is 500
		ChunkStart int
		// total size of result rows (optional)
		TotalLength int
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
