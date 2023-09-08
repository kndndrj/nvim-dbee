package conn

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

	// History is required to act as an input, output and provide a List method
	History interface {
		Output
		Input
		Layout() ([]models.Layout, error)
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

type CallState int

const (
	CallStateExecuting CallState = iota
	CallStateCaching
	CallStateCached
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

// Call represents a single call to database
// it contains various metadata fields, state and a context cancelation function
type Call struct {
	ID     string
	Query  string
	State  CallState
	Cancel func()
}

type Conn struct {
	driver Client
	cache  *cache
	// how many results to wait for in the main thread? -> see cache.go
	blockUntil int
	history    History
	log        models.Logger
	// id of the current call
	currentCallID string
	// map of call stats
	calls map[string]*Call
}

func New(driver Client, blockUntil int, history History, logger models.Logger) *Conn {
	return &Conn{
		blockUntil: blockUntil,
		driver:     driver,
		history:    history,
		cache:      NewCache(blockUntil, logger),
		log:        logger,
		calls:      make(map[string]*Call),
	}
}

// lists call statistics
func (c *Conn) ListCalls() []*Call {
	var calls []*Call

	for _, c := range c.calls {
		calls = append(calls, c)
	}

	return calls
}

func (c *Conn) GetCall(callID string) *Call {
	return c.calls[callID]
}

func (c *Conn) Execute(callID string, query string) error {
	c.log.Debugf("executing query: %q", query)

	if callID == "" {
		callID = uuid.New().String()
	}

	// create a new call
	call := &Call{
		ID:    callID,
		Query: query,
		State: CallStateExecuting,
	}
	ctx, cancel := newCallContext(call)
	call.Cancel = func() {
		cancel()
		call.State = CallStateCanceled
	}

	c.currentCallID = callID
	c.calls[callID] = call

	rows, err := c.driver.Query(ctx, query)
	if err != nil {
		call.State = CallStateFailed
		return err
	}

	return c.setResultToCache(ctx, rows, call, true)
}

func (c *Conn) setResultToCache(ctx context.Context, rows models.IterResult, call *Call, fresh bool) error {
	// set new record in cache
	err := c.cache.Set(ctx, rows, c.blockUntil, call.ID)
	if err != nil {
		call.State = CallStateFailed
		return err
	}
	// check if context is still valid
	if err := ctx.Err(); err != nil {
		return err
	}

	if !fresh {
		return nil
	}

	// save record to history
	go func() {
		_, err := c.cache.Get(ctx, call.ID, 0, -1, c.history)
		if err != nil {
			c.log.Debugf("failed flushing result to history: %s", err.Error())
		}

		// TODO: delete old record from cache
	}()

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

// GetResult pipes the selected range of rows to the outputs
// returns length of the result set
func (c *Conn) GetResult(callID string, from int, to int, outputs ...Output) (int, error) {
	ctx := context.TODO()

	call, ok := c.calls[callID]
	if !ok {
		return 0, fmt.Errorf("call with id %q doesn't exist", callID)
	}

	return c.cache.Get(ctx, call.ID, from, to, outputs...)
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

	// history
	history, err := c.history.Layout()
	if err != nil {
		return nil, err
	}
	if len(history) > 0 {
		layout = append(layout, models.Layout{
			Name:              "history",
			Type:              models.LayoutTypeNone,
			ChildrenSortOrder: models.LayourtSortOrderDescending,
			Children:          history,
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
