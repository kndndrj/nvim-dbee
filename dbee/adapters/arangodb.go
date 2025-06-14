package adapters

import (
	"encoding/gob"
	"fmt"
	"net/url"
	"strings"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Register client
func init() {
	_ = register(&Arango{}, "arango", "arangodb")

	gob.Register(&arangoResponse{})
}

var _ core.Adapter = (*Arango)(nil)

type Arango struct{}

// ArangoDB connection string format
// http://<username>:<password>@<hostname>:<port>/_db/<database>?
func (p *Arango) Connect(rawUrl string) (core.Driver, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unexpected scheme: %q", u.Scheme)
	}
	insecureSkipVerify := u.Query().Get("insecure_skip_verify") == "true"
	dbUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	endpoint := connection.NewRoundRobinEndpoints([]string{dbUrl})
	conn := connection.NewHttpConnection(connection.DefaultHTTPConfigurationWrapper(endpoint, insecureSkipVerify))
	if u.Scheme == "https" && u.Query().Get("use_http2") == "true" {
		conn = connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, insecureSkipVerify))
	}

	if u.User != nil {
		// Basic Authentication
		username := u.User.Username()
		isRootUser := username == "root"
		allowEmptyRootPassword := u.Query().Get("allow_empty_root_password") == "true"
		password, ok := u.User.Password()
		if !ok && !(isRootUser && allowEmptyRootPassword) {
			return nil, fmt.Errorf("arango: missing password")
		}
		auth := connection.NewJWTAuthWrapper(username, password)
		conn = auth(conn)
	}

	conn = connection.NewConnectionAsyncWrapper(conn)
	dbName := parseDatabaseNameFromPath(u.Path)

	// Create a client
	client := arangodb.NewClient(conn)
	return &arangoDriver{
		c:      client,
		dbName: dbName,
	}, nil
}

func parseDatabaseNameFromPath(s string) string {
	if s == "" {
		return "_system"
	}
	pathParts := strings.Split(s, "/")
	lastPartWasDb := false
	for _, part := range pathParts {
		if lastPartWasDb {
			return part
		}
		if part == "_db" {
			lastPartWasDb = true
		}
	}

	return "_system"
}

// Arango helpers will be different as they require http to get collections
func (*Arango) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List": fmt.Sprintf(
			"for n in %s limit 500 return n",
			opts.Table,
		),
	}
}
