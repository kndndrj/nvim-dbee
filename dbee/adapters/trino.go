package adapters

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/trinodb/trino-go-client/trino"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func init() {
	_ = register(&Trino{}, "trino")
}

var _ core.Adapter = (*Trino)(nil)

type Trino struct{}

// Connect parses the connectionURL and returns a new core.Driver.
// connectionURL is a DSN in the format of:
//
//	http[s]://[user]@[host]:[port]?catalog=[catalog]&accessToken=[token]
//
// The 'catalog' and 'accessToken' parameters are required. 'accessToken' is
// normally an env or exec template, expanded by dbee before it reaches this
// function:
//
//	?accessToken={{ env "TRINO_TOKEN" }}
//	?accessToken={{ exec "/usr/local/bin/forge auth okta --domain preprod" }}
//
// Any command printing a bare JWT works: the coordinator advertises
// `Bearer realm="Trino", token_type="JWT"` alongside its OAuth2 challenge, so a
// plain bearer token is accepted and the OAuth2 flow can be skipped entirely.
// Note the token is expanded once, when the connection is created, so a
// short-lived JWT will outlive its connection.
//
// The optional 'schemas' parameter is a comma separated allowlist of schemas to
// show in the drawer. It belongs to this adapter and is removed before the URL
// is handed to the trino driver. Every other parameter is passed through, see
// https://github.com/trinodb/trino-go-client for the full list.
func (t *Trino) Connect(rawURL string) (core.Driver, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	query := parsedURL.Query()

	catalog := query.Get("catalog")
	if catalog == "" {
		return nil, fmt.Errorf("required parameter '?catalog=<catalog>' is missing")
	}

	token := query.Get("accessToken")
	if token == "" {
		return nil, fmt.Errorf("required parameter '?accessToken=<token>' is missing")
	}
	// a leftover template means the exec/env expansion did not run - without this
	// check the user gets an opaque 401 on their first query instead.
	if strings.Contains(token, "{{") {
		return nil, fmt.Errorf("accessToken was not expanded: check the exec or env template in the connection url")
	}

	// 'schemas' is ours, the trino driver must not see it.
	schemas := parseSchemas(query.Get("schemas"))
	query.Del("schemas")
	parsedURL.RawQuery = query.Encode()

	db, err := sql.Open("trino", parsedURL.String())
	if err != nil {
		return nil, fmt.Errorf("invalid trino connection string: %w", err)
	}

	// NOTE: no ping here. Trino has no free ping and a "SELECT 1" on every startup
	// is not worth its cost - connection errors surface on the first query.
	return &trinoDriver{
		c:              builders.NewClient(db),
		connectionURL:  parsedURL,
		catalog:        catalog,
		schemas:        schemas,
		schemasCatalog: catalog,
	}, nil
}

// quoteTrino joins the given parts into a dotted identifier, quoting each part
// and doubling any quote inside it: ("a", `we"ird`) -> `"a"."we""ird"`.
func quoteTrino(parts ...string) string {
	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = `"` + strings.ReplaceAll(part, `"`, `""`) + `"`
	}
	return strings.Join(quoted, ".")
}

// GetHelpers returns a map of helper queries for the given table.
//
// The queries are two part and resolve against the catalog the DSN selects,
// because core.TableOptions carries no catalog and the Trino adapter value is a
// stateless singleton shared by every connection.
//
// The four metadata table helpers only exist on Iceberg and Hive tables and
// error elsewhere. They are offered unconditionally, because learning the
// connector type costs another metadata round trip and a failed helper costs one
// error message.
func (t *Trino) GetHelpers(opts *core.TableOptions) map[string]string {
	table := quoteTrino(opts.Schema, opts.Table)

	metadata := func(suffix string) string {
		return fmt.Sprintf("SELECT * FROM %s;", quoteTrino(opts.Schema, opts.Table+suffix))
	}

	return map[string]string{
		"List":       fmt.Sprintf("SELECT * FROM %s LIMIT 100;", table),
		"Columns":    fmt.Sprintf("DESCRIBE %s;", table),
		"DDL":        fmt.Sprintf("SHOW CREATE TABLE %s;", table),
		"Statistics": fmt.Sprintf("SHOW STATS FOR %s;", table),
		"Partitions": metadata("$partitions"),
		"Snapshots":  metadata("$snapshots"),
		"History":    metadata("$history"),
		"Files":      metadata("$files"),
	}
}

// parseSchemas splits a comma separated schema allowlist, ignoring blanks.
func parseSchemas(raw string) []string {
	var out []string
	for _, schema := range strings.Split(raw, ",") {
		if schema = strings.TrimSpace(schema); schema != "" {
			out = append(out, schema)
		}
	}
	return out
}
