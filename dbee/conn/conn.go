package conn

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
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
)

type ID string

type Conn struct {
	ID   ID
	Name string
	Type string
	URL  string

	driver Client
	log    models.Logger
}

type Params struct {
	ID   ID
	Name string
	Type string
	URL  string
}

func New(params *Params, driver Client, logger models.Logger) *Conn {
	id := params.ID
	if id == "" {
		id = ID(uuid.New().String())
	}

	c := &Conn{
		ID:   id,
		Name: params.Name,
		Type: params.Type,
		URL:  params.URL,

		driver: driver,
		log:    logger,
	}

	return c
}

func (c *Conn) Execute(query string, onEvent func(state call.State)) *call.Stat {
	c.log.Debugf("executing query: %q", query)

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

func (c *Conn) Structure() ([]models.Layout, error) {
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
