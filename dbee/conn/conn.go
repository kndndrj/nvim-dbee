package conn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/neovim/go-client/msgpack"
)

var ErrDatabaseSwitchingNotSupported = errors.New("database switching not supported")

type (
	// Client is an interface for a specific database driver
	Client interface {
		Query(context.Context, string) (models.IterResult, error)
		Layout() ([]models.Layout, error)
		Close()
	}

	// DatabaseSwitcher is an optional interface for clients that have database switching capabilities
	DatabaseSwitcher interface {
		SelectDatabase(string) error
		ListDatabases() (current string, available []string, err error)
	}

	Adapter interface {
		Connect(typ string, url string) (Client, error)
	}
)

type ID string

type Params struct {
	ID   ID
	Name string
	Type string
	URL  string
}

// Expand returns a copy of the original parameters with expanded fields
func (p *Params) Expand() *Params {
	return &Params{
		ID:   ID(expand(string(p.ID))),
		Name: expand(p.Name),
		Type: expand(p.Type),
		URL:  expand(p.URL),
	}
}

type paramsPersistent struct {
	ID   string `msgpack:"id" json:"id"`
	Name string `msgpack:"name" json:"name"`
	Type string `msgpack:"type" json:"type"`
	URL  string `msgpack:"url" json:"url"`
}

func (s *Params) toPersistent() *paramsPersistent {
	return &paramsPersistent{
		ID:   string(s.ID),
		Name: s.Name,
		Type: s.Type,
		URL:  s.URL,
	}
}

func (p *Params) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(p.toPersistent())
}

func (p *Params) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.toPersistent())
}

type Conn struct {
	params           *Params
	unexpandedParams *Params

	driver Client
}

func (s *Conn) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(s.params)
}

func (s *Conn) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.params)
}

func New(params *Params, adapter Adapter) (*Conn, error) {
	expanded := params.Expand()

	if expanded.ID == "" {
		expanded.ID = ID(uuid.New().String())
	}

	driver, err := adapter.Connect(expanded.Type, expanded.URL)
	if err != nil {
		return nil, fmt.Errorf("adapter.Connect: %w", err)
	}

	c := &Conn{
		params:           expanded,
		unexpandedParams: params,

		driver: driver,
	}

	return c, nil
}

func (c *Conn) GetID() ID {
	return c.params.ID
}

func (c *Conn) GetName() string {
	return c.params.Name
}

func (c *Conn) GetType() string {
	return c.params.Type
}

func (c *Conn) GetURL() string {
	return c.params.URL
}

// GetParams returns the original source for this connection
func (c *Conn) GetParams() *Params {
	return c.unexpandedParams
}

func (c *Conn) Execute(query string, onEvent func(state call.State)) *call.Stat {
	exec := func(ctx context.Context) (models.IterResult, error) {
		return c.driver.Query(ctx, query)
	}

	return call.NewStatFromExecutor(exec, query, onEvent)
}

// SelectDatabase tries to switch to a given database with the used client.
// on error, the switch doesn't happen and the previous connection remains active.
func (c *Conn) SelectDatabase(name string) error {
	switcher, ok := c.driver.(DatabaseSwitcher)
	if !ok {
		return ErrDatabaseSwitchingNotSupported
	}

	err := switcher.SelectDatabase(name)
	if err != nil {
		return fmt.Errorf("switcher.SelectDatabase: %w", err)
	}

	return nil
}

func (c *Conn) ListDatabases() (current string, available []string, err error) {
	switcher, ok := c.driver.(DatabaseSwitcher)
	if !ok {
		return "", nil, ErrDatabaseSwitchingNotSupported
	}

	currentDB, availableDBs, err := switcher.ListDatabases()
	if err != nil {
		return "", nil, fmt.Errorf("switcher.ListDatabases: %w", err)
	}

	return currentDB, availableDBs, nil
}

func (c *Conn) GetStructure() ([]models.Layout, error) {
	// structure
	structure, err := c.driver.Layout()
	if err != nil {
		return nil, err
	}

	// fallback to not confuse users
	if len(structure) < 1 {
		structure = []models.Layout{
			{
				Name: "no schema to show",
				Type: models.LayoutTypeNone,
			},
		}
	}
	return structure, nil
}

func (c *Conn) Close() {
	c.driver.Close()
}
