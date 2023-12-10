package handler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/neovim/go-client/nvim"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/format"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

const callLogFileName = "/tmp/dbee-calllog.json"

type Handler struct {
	vim    *nvim.Nvim
	log    *vim.Logger
	events *eventBus

	lookupConnection     map[core.ConnectionID]*core.Connection
	lookupCall           map[core.CallID]*core.Call
	lookupConnectionCall map[core.ConnectionID][]core.CallID

	currentConnectionID core.ConnectionID
}

func NewHandler(vim *nvim.Nvim, logger *vim.Logger) *Handler {
	h := &Handler{
		vim: vim,
		log: logger,
		events: &eventBus{
			vim: vim,
			log: logger,
		},

		lookupConnection:     make(map[core.ConnectionID]*core.Connection),
		lookupCall:           make(map[core.CallID]*core.Call),
		lookupConnectionCall: make(map[core.ConnectionID][]core.CallID),
	}

	// restore the call log concurrently
	go func() {
		err := h.restoreCallLog()
		if err != nil {
			h.log.Debugf("h.restoreCallLog: %s", err)
		}
	}()

	return h
}

func (h *Handler) Close() {
	err := h.storeCallLog()
	if err != nil {
		h.log.Debugf("h.storeCallLog: %s", err)
	}

	for _, c := range h.lookupConnection {
		c.Close()
	}
}

func (h *Handler) CreateConnection(params *core.ConnectionParams) (core.ConnectionID, error) {
	c, err := adapters.NewConnection(params)
	if err != nil {
		return "", fmt.Errorf("core.New: %w", err)
	}

	old, ok := h.lookupConnection[c.GetID()]
	if ok {
		go old.Close()
	}

	h.lookupConnection[c.GetID()] = c
	_ = h.SetCurrentConnection(c.GetID())

	return c.GetID(), nil
}

func (h *Handler) GetConnections(ids []core.ConnectionID) []*core.Connection {
	var conns []*core.Connection

	for _, c := range h.lookupConnection {
		if len(ids) > 0 && !slices.Contains(ids, c.GetID()) {
			continue
		}
		conns = append(conns, c)
	}

	return conns
}

func (h *Handler) GetCurrentConnection() (*core.Connection, error) {
	c, ok := h.lookupConnection[h.currentConnectionID]
	if !ok {
		return nil, fmt.Errorf("current connection has not been set yet")
	}
	return c, nil
}

func (h *Handler) SetCurrentConnection(connID core.ConnectionID) error {
	_, ok := h.lookupConnection[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	if h.currentConnectionID == connID {
		return nil
	}

	// update connection and trigger event
	h.currentConnectionID = connID
	h.events.CurrentConnectionChanged(connID)

	return nil
}

func (h *Handler) ConnectionExecute(connID core.ConnectionID, query string) (*core.Call, error) {
	c, ok := h.lookupConnection[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	call := c.Execute(query, func(cl *core.Call) {
		if err := cl.Err(); err != nil {
			h.log.Errorf("cl.Err: %s", err)
		}

		h.events.CallStateChanged(cl)
	})

	id := call.GetID()

	// add to lookup
	h.lookupCall[id] = call
	h.lookupConnectionCall[connID] = append(h.lookupConnectionCall[connID], id)

	// update current call and conn
	_ = h.SetCurrentConnection(connID)

	return call, nil
}

func (h *Handler) ConnectionGetCalls(connID core.ConnectionID) ([]*core.Call, error) {
	_, ok := h.lookupConnection[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	var calls []*core.Call
	callIDs, ok := h.lookupConnectionCall[connID]
	if !ok {
		return calls, nil
	}
	for _, cID := range callIDs {
		c, ok := h.lookupCall[cID]
		if !ok {
			continue
		}
		calls = append(calls, c)
	}

	return calls, nil
}

func (h *Handler) ConnectionGetParams(connID core.ConnectionID) (*core.ConnectionParams, error) {
	c, ok := h.lookupConnection[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	return c.GetParams(), nil
}

func (h *Handler) ConnectionGetStructure(connID core.ConnectionID) ([]*core.Structure, error) {
	c, ok := h.lookupConnection[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	layout, err := c.GetStructure()
	if err != nil {
		return nil, fmt.Errorf("c.GetStructure: %w", err)
	}

	return layout, nil
}

func (h *Handler) ConnectionListDatabases(connID core.ConnectionID) (current string, available []string, err error) {
	c, ok := h.lookupConnection[connID]
	if !ok {
		return "", nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	currentDB, availableDBs, err := c.ListDatabases()
	if err != nil {
		if errors.Is(err, core.ErrDatabaseSwitchingNotSupported) {
			return "", []string{}, nil
		}
		return "", nil, fmt.Errorf("c.ListDatabases: %w", err)
	}

	return currentDB, availableDBs, nil
}

func (h *Handler) ConnectionSelectDatabase(connID core.ConnectionID, database string) error {
	c, ok := h.lookupConnection[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	err := c.SelectDatabase(database)
	if err != nil {
		return fmt.Errorf("c.SelectDatabase: %w", err)
	}

	return nil
}

func (h *Handler) CallCancel(callID core.CallID) error {
	call, ok := h.lookupCall[callID]
	if !ok {
		return fmt.Errorf("unknown call with id: %q", callID)
	}

	call.Cancel()
	return nil
}

func (h *Handler) CallDisplayResult(callID core.CallID, buffer nvim.Buffer, from, to int) (int, error) {
	call, ok := h.lookupCall[callID]
	if !ok {
		return 0, fmt.Errorf("unknown call with id: %q", callID)
	}

	res, err := call.GetResult()
	if err != nil {
		return 0, fmt.Errorf("call.GetResult: %w", err)
	}

	text, err := res.Format(newTable(), from, to)
	if err != nil {
		return 0, fmt.Errorf("res.Format: %w", err)
	}

	_, err = newBuffer(h.vim, buffer).Write(text)
	if err != nil {
		return 0, fmt.Errorf("buffer.Write: %w", err)
	}

	return res.Len(), nil
}

func (h *Handler) CallStoreResult(callID core.CallID, fmat, out string, from, to int, arg ...any) error {
	stat, ok := h.lookupCall[callID]
	if !ok {
		return fmt.Errorf("unknown call with id: %q", callID)
	}

	var formatter core.Formatter
	switch fmat {
	case "json":
		formatter = format.NewJSON()
	case "csv":
		formatter = format.NewCSV()
	case "table":
		formatter = newTable()
	default:
		return fmt.Errorf("store output: %q is not supported", fmat)
	}

	var writer io.Writer
	switch out {
	case "file":
		if len(arg) < 1 || arg[0] == "" {
			return fmt.Errorf("invalid output path")
		}
		path, ok := arg[0].(string)
		if !ok {
			return fmt.Errorf("invalid output path")
		}

		writer, err := os.Create(path)
		if err != nil {
			return err
		}
		defer writer.Close()
	case "buffer":
		if len(arg) < 1 {
			return fmt.Errorf("invalid output path")
		}
		buf, ok := arg[0].(int)
		if !ok {
			return fmt.Errorf("invalid output path")
		}
		writer = newBuffer(h.vim, nvim.Buffer(buf))
	case "yank":
		writer = newYankRegister(h.vim)
	default:
		return fmt.Errorf("store output: %q is not supported", out)
	}

	res, err := stat.GetResult()
	if err != nil {
		return fmt.Errorf("stat.GetResult: %w", err)
	}

	text, err := res.Format(formatter, from, to)
	if err != nil {
		return fmt.Errorf("res.Format: %w", err)
	}

	_, err = writer.Write(text)
	if err != nil {
		return fmt.Errorf("buffer.Write: %w", err)
	}

	return nil
}
