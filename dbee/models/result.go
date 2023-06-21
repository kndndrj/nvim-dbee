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
		Query      string
		Timestamp  time.Time
		SchemaType SchemaType
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
