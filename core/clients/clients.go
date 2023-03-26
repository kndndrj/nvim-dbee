package clients

type Row []any
type Header []string

type Schema map[string][]string

type Rows interface {
	Header() (Header, error)
	Next() (Row, error)
	Close()
}

type Client interface {
	Execute(query string) (Rows, error)
	Close()
	Schema() (Schema, error)
}
