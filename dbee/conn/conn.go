package conn

import (
	"context"
	"fmt"

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
	id     string
	driver Client
	// how many results to wait for in the main thread? -> see cache.go
	log models.Logger
	// map of call stats
	calls map[string]*call.Call
}

func New(id string, driver Client, logger models.Logger) *Conn {
	c := &Conn{
		id:     id,
		driver: driver,
		log:    logger,
		calls:  make(map[string]*call.Call),
	}

	go c.scanOldCalls()

	return c
}

func (c *Conn) scanOldCalls() {
	calls := ScanOldCalls(c.id, c.log)

	for _, ca := range calls {
		c.calls[ca.GetDetails().ID] = ca
	}
}

func (c *Conn) archiveCall(ca *call.Call) {
	err := ArchiveCall(c.id, ca)
	if err != nil {
		c.log.Debugf("failed archiving call details: %s", ca.GetDetails().ID)
	}
}

func (c *Conn) Calls() []*call.CallDetails {
	var calls []*call.CallDetails

	for _, c := range c.calls {
		calls = append(calls, c.GetDetails())
	}

	return calls
}

func (c *Conn) Execute(query string, callback func(*call.CallDetails)) (string, error) {
	c.log.Debugf("executing query: %q", query)

	exec := func(ctx context.Context) (models.IterResult, error) {
		return c.driver.Query(ctx, query)
	}

	ca := call.NewCaller(c.log).
		WithExecutor(exec).
		WithCallback(callback).
		WithQuery(query).
		Do()

	c.calls[ca.GetDetails().ID] = ca

	go c.archiveCall(ca)

	return ca.GetDetails().ID, nil
}

func (c *Conn) CancelCall(callID string) {
	ca, ok := c.calls[callID]
	if !ok {
		return
	}

	ca.Cancel()
}

func (c *Conn) GetResult(callID string, from int, to int, outputs ...call.Output) (int, error) {
	ca, ok := c.calls[callID]
	if !ok {
		return 0, fmt.Errorf("no call with id: %q", callID)
	}

	return ca.GetResult(from, to, outputs...)
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
