package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/neovim/go-client/nvim"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/format"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

const callLogFileName = "/tmp/dbee-calllog.json"

type eventBus struct {
	vim *nvim.Nvim
	log *vim.Logger
}

func (eb *eventBus) callLua(event string, data string) {
	err := eb.vim.ExecLua(fmt.Sprintf(`require("dbee.handler.__callbacks").trigger(%q, %s)`, event, data), nil)
	if err != nil {
		eb.log.Debugf("eb.vim.ExecLua: %s", err)
	}
}

func (eb *eventBus) CallStateChanged(call *core.Stat) {
	data := fmt.Sprintf(`{
		call = {
			id = %q,
			query = %q,
			state = %q,
			took_us = %d,
			timestamp_us = %d,
		},
	}`, call.ID,
		call.Query,
		call.State.String(),
		call.Took.Microseconds(),
		call.Timestamp.UnixMicro())

	eb.callLua("call_state_changed", data)
}

func (eb *eventBus) CurrentConnectionChanged(id core.ID) {
	data := fmt.Sprintf(`{
		conn_id = %q,
	}`, id)

	eb.callLua("current_connection_changed", data)
}

type Handler struct {
	vim    *nvim.Nvim
	log    *vim.Logger
	events *eventBus

	lookupConn     map[core.ID]*core.Conn
	lookupStat     map[core.StatID]*core.Stat
	lookupConnStat map[core.ID][]core.StatID

	currentConnID core.ID
	currentStatID core.StatID
}

func NewHandler(vim *nvim.Nvim, logger *vim.Logger) *Handler {
	h := &Handler{
		vim: vim,
		log: logger,
		events: &eventBus{
			vim: vim,
			log: logger,
		},

		lookupConn:     make(map[core.ID]*core.Conn),
		lookupStat:     make(map[core.StatID]*core.Stat),
		lookupConnStat: make(map[core.ID][]core.StatID),
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

func (h *Handler) storeCallLog() error {
	store := make(map[core.ID][]*core.Stat)

	for connID := range h.lookupConn {
		calls, err := h.ConnGetCalls(connID)
		if err != nil || len(calls) < 1 {
			continue
		}
		store[connID] = calls
	}

	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("json.MarshalIndent: %w", err)
	}

	file, err := os.Create(callLogFileName)
	if err != nil {
		return fmt.Errorf("os.Create: %s", err)
	}
	defer file.Close()

	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("file.Write: %w", err)
	}

	return nil
}

func (h *Handler) restoreCallLog() error {
	file, err := os.Open(callLogFileName)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var store map[core.ID][]*core.Stat

	err = decoder.Decode(&store)
	if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	for connID, calls := range store {
		callIDs := make([]core.StatID, len(calls))

		// fill call lookup
		for i, c := range calls {
			h.lookupStat[c.ID] = c
			callIDs[i] = c.ID
		}

		// add to conn-call lookup
		h.lookupConnStat[connID] = append(h.lookupConnStat[connID], callIDs...)
	}

	return nil
}

func (h *Handler) Close() {
	err := h.storeCallLog()
	if err != nil {
		h.log.Debugf("h.storeCallLog: %s", err)
	}

	for _, c := range h.lookupConn {
		c.Close()
	}
}

func (h *Handler) CreateConnection(params *core.Params) (core.ID, error) {
	c, err := core.New(params, clients.Adapter())
	if err != nil {
		return "", fmt.Errorf("core.New: %w", err)
	}

	old, ok := h.lookupConn[c.GetID()]
	if ok {
		go old.Close()
	}

	h.lookupConn[c.GetID()] = c

	return c.GetID(), nil
}

func (h *Handler) DeleteConnection(connID core.ID) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return
	}
	c.Close()
	delete(h.lookupConn, connID)
}

func (h *Handler) GetConnections(ids []core.ID) []*core.Conn {
	var conns []*core.Conn

	for _, c := range h.lookupConn {
		if len(ids) > 0 && !slices.Contains(ids, c.GetID()) {
			continue
		}
		conns = append(conns, c)
	}

	return conns
}

func (h *Handler) GetCurrentConnection() (*core.Conn, error) {
	c, ok := h.lookupConn[h.currentConnID]
	if !ok {
		return nil, fmt.Errorf("current connection has not been set yet")
	}
	return c, nil
}

func (h *Handler) SetCurrentConnection(connID core.ID) error {
	_, ok := h.lookupConn[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	fmt.Println("here")

	if h.currentConnID == connID {
		return nil
	}

	// update connection and trigger event
	h.currentConnID = connID
	h.events.CurrentConnectionChanged(connID)

	return nil
}

func (h *Handler) ConnExecute(connID core.ID, query string) (*core.Stat, error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	stat := new(core.Stat)
	onEvent := func(state core.State) {
		h.events.CallStateChanged(stat)
	}

	stat = c.Execute(query, onEvent)

	id := stat.ID

	// add to lookup
	h.lookupStat[id] = stat
	h.lookupConnStat[connID] = append(h.lookupConnStat[connID], id)

	// update current call and conn
	h.currentStatID = id
	_ = h.SetCurrentConnection(connID)

	return stat, nil
}

func (h *Handler) ConnGetCalls(connID core.ID) ([]*core.Stat, error) {
	_, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	var calls []*core.Stat
	callIDs, ok := h.lookupConnStat[connID]
	if !ok {
		return calls, nil
	}
	for _, cID := range callIDs {
		c, ok := h.lookupStat[cID]
		if !ok {
			continue
		}
		calls = append(calls, c)
	}

	return calls, nil
}

func (h *Handler) ConnGetParams(connID core.ID) (*core.Params, error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	return c.GetParams(), nil
}

func (h *Handler) ConnGetStructure(connID core.ID) ([]core.Layout, error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	layout, err := c.GetStructure()
	if err != nil {
		return nil, fmt.Errorf("c.GetStructure: %w", err)
	}

	return layout, nil
}

func (h *Handler) ConnListDatabases(connID core.ID) (current string, available []string, err error) {
	c, ok := h.lookupConn[connID]
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

func (h *Handler) ConnSelectDatabase(connID core.ID, database string) error {
	c, ok := h.lookupConn[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	err := c.SelectDatabase(database)
	if err != nil {
		return fmt.Errorf("c.SelectDatabase: %w", err)
	}

	return nil
}

func (h *Handler) CallCancel(callID core.StatID) error {
	stat, ok := h.lookupStat[callID]
	if !ok {
		return fmt.Errorf("unknown call with id: %q", callID)
	}

	stat.Cancel()
	return nil
}

func (h *Handler) CallDisplayResult(callID core.StatID, buffer nvim.Buffer, from, to int) (int, error) {
	stat, ok := h.lookupStat[callID]
	if !ok {
		return 0, fmt.Errorf("unknown call with id: %q", callID)
	}

	res, err := stat.GetResult()
	if err != nil {
		return 0, fmt.Errorf("stat.GetResult: %w", err)
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

func (h *Handler) CallStoreResult(callID core.StatID, fmat, out string, from, to int, arg ...any) error {
	stat, ok := h.lookupStat[callID]
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
