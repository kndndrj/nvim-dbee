package conn

import (
	"context"
	"encoding/json"
	"errors"
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
	CallStateUninitialized CallState = iota
	CallStateExecuting
	CallStateCaching
	CallStateCached
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

// Call represents a single call to database
// it contains various metadata fields, state and a context cancelation function
type Call struct {
	id      string
	state   CallState
	cancel  func()
	cache   *cache
	history *HistoryOutput
	log     models.Logger
}

func makeCall(id string, connID string, logger models.Logger, exec func(context.Context) (models.IterResult, error)) *Call {
	c := &Call{
		id:      id,
		state:   CallStateUninitialized,
		cache:   NewCache(),
		history: NewHistory(connID),
		log:     logger,
	}

	// drain the exec function in a separate coroutine
	go func() {
		err := c.Do(exec)
		if err != nil {
			c.log.Error(err.Error())
		}
	}()

	return c
}

func (c *Call) Do(exec func(context.Context) (models.IterResult, error)) error {
	ctx, cancel := context.WithCancel(context.Background())

	if c.cancel != nil {
		oldCancel := c.cancel
		cancel = func() {
			oldCancel()
			cancel()
		}
	}
	c.cancel = cancel

	c.state = CallStateExecuting
	rows, err := exec(ctx)
	if err != nil {
		c.state = CallStateFailed
		if errors.Is(err, context.Canceled) {
			c.state = CallStateCanceled
		}
		return err
	}

	// save to cache
	c.state = CallStateCaching
	err = c.cache.Set(ctx, rows)
	if err != nil && !errors.Is(err, ErrAlreadyFilled) {
		c.state = CallStateFailed
		if errors.Is(err, context.Canceled) {
			c.state = CallStateCanceled
		}
		return err
	}

	// save to history
	c.state = CallStateCached
	_, err = c.cache.Get(ctx, 0, -1, c.history)
	if err != nil && !errors.Is(err, ErrAlreadyFilled) {
		return err
	}
	c.state = CallStateArchived

	return nil
}

// GetResult pipes the selected range of rows to the outputs
// returns length of the result set
func (c *Call) GetResult(from int, to int, outputs ...Output) (int, error) {
	if !c.cache.HasResult() && c.history.HasResult() {
		go func() {
			err := c.Do(c.history.Query)
			if err != nil {
				c.log.Error(err.Error())
			}
		}()
	}

	ctx := context.TODO()

	return c.cache.Get(ctx, from, to, outputs...)
}

func (c *Call) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (s *Call) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID    string `json:"id"`
		State int    `json:"state"`
	}{
		ID:    s.id,
		State: int(s.state),
	})
}

type Conn struct {
	driver Client
	// how many results to wait for in the main thread? -> see cache.go
	log models.Logger
	// id of the current call
	currentCallID string
	// map of call stats
	calls map[string]*Call
}

func New(driver Client, logger models.Logger) *Conn {
	return &Conn{
		driver: driver,
		log:    logger,
		calls:  make(map[string]*Call),
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
	if callID == "" {
		callID = c.currentCallID
	}
	return c.calls[callID]
}

func (c *Conn) Execute(callID string, query string) error {
	c.log.Debugf("executing query: %q", query)

	if callID == "" {
		callID = uuid.New().String()
	}

	c.calls[callID] = makeCall(callID, "TODO", c.log, func(ctx context.Context) (models.IterResult, error) {
		return c.driver.Query(ctx, query)
	})
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
