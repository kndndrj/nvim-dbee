package conn

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type (
	// Input requires implementaions to provide iterator from a
	// given string input, which can be a query or some sort of id
	Input interface {
		Query(context.Context, string) (models.IterResult, error)
	}

	// Output recieves a result and does whatever it wants with it
	Output interface {
		Write(context.Context, models.Result) error
	}

	// Client is a special kind of input with extra stuff
	Client interface {
		Input
		Close()
		Layout() ([]models.Layout, error)
	}

	// DatabaseSwitcher is an optional interface for clients that have database switching capabilities
	DatabaseSwitcher interface {
		SelectDatabase(string) error
		ListDatabases() (current string, available []string, err error)
	}
)

type Conn struct {
	driver Client
	// how many results to wait for in the main thread? -> see cache.go
	log models.Logger
	// id of the current call
	currentCallID string
	// map of call stats
	calls map[string]*call.Call
}

func New(driver Client, logger models.Logger) *Conn {
	return &Conn{
		driver: driver,
		log:    logger,
		calls:  make(map[string]*call.Call),
	}
}

// lists call statistics
func (c *Conn) ListCalls() []*call.Call {
	var calls []*call.Call

	for _, c := range c.calls {
		calls = append(calls, c)
	}

	return calls
}

func (c *Conn) GetCall(callID string) (*call.Call, bool) {
	if callID == "" {
		callID = c.currentCallID
	}

	ca, ok := c.calls[callID]
	return ca, ok
}

func (c *Conn) Execute(callID string, query string) error {
	c.log.Debugf("executing query: %q", query)

	if callID == "" {
		callID = uuid.New().String()
	}

	ca := call.MakeCall(callID, "TODO", c.log)
	err := ca.Do(func(ctx context.Context) (models.IterResult, error) {
		return c.driver.Query(ctx, query)
	})
	if err != nil {
		return err
	}

	c.calls[callID] = ca
	c.currentCallID = callID

	return nil
}

// SwitchDatabase tries to switch to a given database with the used client.
// on error, the switch doesn't happen and the previous connection remains active.
func (c *Conn) SwitchDatabase(name string) error {
	switcher, ok := c.driver.(DatabaseSwitcher)
	if !ok {
		return fmt.Errorf("connection does not support database switching")
	}

	err := switcher.SelectDatabase(name)
	if err != nil {
		return fmt.Errorf("failed to switch to different database: %w", err)
	}

	return nil
}

func (c *Conn) Layout() ([]models.Layout, error) {
	var layout []models.Layout

	// structure
	structure, err := c.driver.Layout()
	if err != nil {
		return nil, err
	}
	if len(structure) > 0 {
		layout = append(layout, models.Layout{
			Name:     "structure",
			Type:     models.LayoutTypeNone,
			Children: structure,
		})
	}

	// databases
	if switcher, ok := c.driver.(DatabaseSwitcher); ok {
		currentDB, availableDBs, err := switcher.ListDatabases()
		if err != nil {
			return nil, err
		}

		layout = append(layout, models.Layout{
			Name:      currentDB,
			Type:      models.LayoutTypeDatabaseSwitch,
			PickItems: availableDBs,
		})
	}

	// fallback to not confuse users
	if len(layout) < 1 {
		layout = append(layout, models.Layout{
			Name: "no schema to show",
			Type: models.LayoutTypeNone,
		})
	}
	return layout, nil
}

func (c *Conn) Close() {
	c.driver.Close()
}
