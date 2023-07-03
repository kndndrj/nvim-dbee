package models

import "time"

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
		// actual query which gave the result
		Query      string
		// timestamp of the executed query
		Timestamp  time.Time
		// type of schema (shcemaful or schemaless)
		SchemaType SchemaType
		// position of the first row of the result - if the result is from row 500 to 1000, this nubmer is 500
		ChunkStart int
	}

	// IterResult is an iterator which provides rows and headers from the Input
	IterResult interface {
		Meta() (Meta, error)
		Header() (Header, error)
		Next() (Row, error)
		Close()
	}

	// Result is the "drained" form of the IterResult iterator used by Output
	Result struct {
		Header Header
		Rows   []Row
		Meta   Meta
	}
)
