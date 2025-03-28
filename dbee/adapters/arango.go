package adapters

import (
	"encoding/gob"
	"fmt"
	"net/url"

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

func (p *Arango) Connect(rawUrl string) (core.Driver, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("arango: invalid url: %w", err)
	}
	// Set authentication
	dbUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	endpoint := connection.NewRoundRobinEndpoints([]string{dbUrl})
	conn := connection.NewHttpConnection(connection.DefaultHTTPConfigurationWrapper(endpoint, true))
	if u.Scheme == "https" {
		conn = connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, false))
	}

	if u.User != nil {
		// Basic Authentication
		username := u.User.Username()
		password, _ := u.User.Password()
		auth := connection.NewJWTAuthWrapper(username, password)
		conn = auth(conn)
	}

	conn = connection.NewConnectionAsyncWrapper(conn)

	// Create a client
	client := arangodb.NewClient(conn)
	return &arangoDriver{
		c:      client,
		dbName: "_system",
	}, nil
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
