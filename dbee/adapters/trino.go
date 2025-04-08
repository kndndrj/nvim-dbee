package adapters

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/kndndrj/nvim-dbee/dbee/core" // Assuming we might need builders later
	"github.com/trinodb/trino-go-client/trino"
)

// Register client
func init() {
	_ = register(&Trino{}, "trino")
}

var (
	_ core.Adapter = (*Trino)(nil)
	// Add DatabaseSwitcher if implemented later
	// _ core.DatabaseSwitcher = (*trinoDriver)(nil)
)

type Trino struct{}

// Connect establishes a connection to a Trino cluster.
// The URL format is expected to be http[s]://user[:password]@host:port?catalog=<catalog>&schema=<schema>&source=<source>...
// See https://github.com/trinodb/trino-go-client for more config options.
func (t *Trino) Connect(connectionURL string) (core.Driver, error) {
	parsedURL, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %q: %w", connectionURL, err)
	}

	cfg := &trino.Config{
		ServerURI: parsedURL.String(),
	}

	client, err := sql.Open("trino", parsedURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create trino connection: %w", err)
	}

	// Store the parsed URL and initial catalog/schema if needed for switching later
	parsedURL, _ = url.Parse(parsedURL.String()) // Ignore error as NewConfig already parsed it

	return &trinoDriver{
		client:        client,
		cfg:           cfg,       // Store config for potential state access (catalog/schema)
		connectionURL: parsedURL, // Store URL for potential reconnection on switching
	}, nil
}

// GetHelpers returns Trino-specific helper queries.
func (t *Trino) GetHelpers(opts *core.TableOptions) map[string]string {
	// Remove qualifiedTable since we can't use catalog
	schemaTable := fmt.Sprintf("%s.%s", opts.Schema, opts.Table)

	return map[string]string{
		"List":        fmt.Sprintf("SELECT * FROM %s LIMIT 100", schemaTable),
		"Columns":     fmt.Sprintf("DESCRIBE %s", schemaTable),
		"Show Create": fmt.Sprintf("SHOW CREATE TABLE %s", schemaTable),
	}
}

// Helper function to extract catalog and schema from URL (if needed outside Connect)
func getCatalogSchemaFromURL(u *url.URL) (string, string) {
	q := u.Query()
	return q.Get("catalog"), q.Get("schema")
}
