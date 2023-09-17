package handler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/kndndrj/nvim-dbee/dbee/output/format"
	"github.com/neovim/go-client/nvim"
)

type eventBus struct {
	vim *nvim.Nvim
}

func (eb *eventBus) callLua(event string, data string) error {
	err := eb.vim.ExecLua(fmt.Sprintf(`require("dbee.handler.__callbacks").trigger(%q, %s)`, event, data), nil)
	if err != nil {
		return fmt.Errorf("eb.vim.ExecLua: %w", err)
	}

	return nil
}

func (eb *eventBus) CallStateChanged(call *call.Stat) error {
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

	// trigger callback
	return eb.callLua("call_state_changed", data)
}

func (eb *eventBus) CurrentConnectionChanged(id conn.ID) error {
	data := fmt.Sprintf(`{
		conn_id = %q,
	}`, id)

	// trigger callback
	return eb.callLua("current_connection_changed", data)
}

type Handler struct {
	vim    *nvim.Nvim
	log    models.Logger
	events *eventBus

	lookupConn     map[conn.ID]*conn.Conn
	lookupStat     map[call.StatID]*call.Stat
	lookupConnStat map[conn.ID][]call.StatID

	currentConnID conn.ID
	currentStatID call.StatID
}

func NewHandler(vim *nvim.Nvim, logger models.Logger) *Handler {
	return &Handler{
		vim: vim,
		log: logger,

		lookupConn:     make(map[conn.ID]*conn.Conn),
		lookupStat:     make(map[call.StatID]*call.Stat),
		lookupConnStat: make(map[conn.ID][]call.StatID),
	}
}

func (h *Handler) CreateConnection(spec *conn.Params) (conn.ID, error) {
	_, ok := h.lookupConn[spec.ID]
	if ok {
		return "", nil
	}

	driver, err := clients.NewFromType(spec.URL, spec.Type)
	if err != nil {
		return "", fmt.Errorf("clients.NewFromType: %w")
	}

	c := conn.New(spec, driver, h.log)

	h.lookupConn[c.ID] = c

	return c.ID, nil
}

func (h *Handler) DeleteConnection(connID conn.ID) {
	delete(h.lookupConn, connID)
}

func (h *Handler) GetConnections(ids []conn.ID) []*conn.Conn {
	var conns []*conn.Conn

	for _, c := range h.lookupConn {
		if len(ids) > 0 && !slices.Contains(ids, c.ID) {
			continue
		}
		conns = append(conns, c)
	}

	return conns
}

func (h *Handler) GetCurrentConnection() (*conn.Conn, error) {
	c, ok := h.lookupConn[h.currentConnID]
	if !ok {
		return nil, fmt.Errorf("current connection has not been set yet")
	}
	return c, nil
}

func (h *Handler) SetCurrentConnection(connID conn.ID) error {
	_, ok := h.lookupConn[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	if h.currentConnID == connID {
		return nil
	}

	// update connection and trigger event
	h.currentConnID = connID
	h.events.CurrentConnectionChanged(connID)

	return nil
}

func (h *Handler) ConnExecute(connID conn.ID, query string) (*call.Stat, error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	stat := new(call.Stat)
	onEvent := func(state call.State) {
		h.events.CallStateChanged(stat)
	}

	stat = c.Execute(query, onEvent)

	id := stat.ID

	// add to lookup
	h.lookupStat[id] = stat
	h.lookupConnStat[connID] = append(h.lookupConnStat[connID], id)

	// update current call and conn
	h.currentStatID = id
	h.SetCurrentConnection(connID)

	return stat, nil
}

func (h *Handler) ConnGetCalls(connID conn.ID) ([]*call.Stat, error) {
	_, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	var calls []*call.Stat
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

func (h *Handler) ConnGetStructure(connID conn.ID) ([]models.Layout, error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	layout, err := c.Structure()
	if err != nil {
		return nil, fmt.Errorf("conn.Layout: %w", err)
	}

	return layout, nil
}

func (h *Handler) ConnListDatabases(connID conn.ID) (current string, available []string, err error) {
	c, ok := h.lookupConn[connID]
	if !ok {
		return "", nil, fmt.Errorf("unknown connection with id: %q", connID)
	}

	currentDB, availableDBs, err := c.ListDatabases()
	if err != nil {
		if errors.Is(err, conn.ErrDatabaseSwitchingNotSupported) {
			return "", []string{}, nil
		}
		return "", nil, fmt.Errorf("conn.ListDatabases: %w", err)
	}

	return currentDB, availableDBs, nil
}

func (h *Handler) ConnSelectDatabase(connID conn.ID, database string) error {
	c, ok := h.lookupConn[connID]
	if !ok {
		return fmt.Errorf("unknown connection with id: %q", connID)
	}

	err := c.SelectDatabase(database)
	if err != nil {
		return fmt.Errorf("conn.SelectDatabase: %w", err)
	}

	return nil
}

func (h *Handler) CallCancel(callID call.StatID) error {
	stat, ok := h.lookupStat[callID]
	if !ok {
		return fmt.Errorf("unknown call with id: %q", callID)
	}

	stat.Cancel()
	return nil
}

func (h *Handler) CallDisplayResult(callID call.StatID, buffer nvim.Buffer, from, to int) (int, error) {
	stat, ok := h.lookupStat[callID]
	if !ok {
		return 0, fmt.Errorf("unknown call with id: %q", callID)
	}

	res, err := stat.GetResult()
	if err != nil {
		return 0, fmt.Errorf("stat.GetResult: %w", err)
	}

	text, err := res.Format(format.NewTable(), from, to)
	if err != nil {
		return 0, fmt.Errorf("res.Format: %w", err)
	}

	_, err = output.NewBuffer(h.vim, buffer).Write(text)
	if err != nil {
		return 0, fmt.Errorf("buffer.Write: %w", err)
	}

	return res.Len(), nil
}

func (h *Handler) CallStoreResult(callID call.StatID, fmat, out string, from, to int, arg ...any) error {
	stat, ok := h.lookupStat[callID]
	if !ok {
		return fmt.Errorf("unknown call with id: %q", callID)
	}

	var formatter call.Formatter
	switch fmat {
	case "json":
		formatter = format.NewJSON()
	case "csv":
		formatter = format.NewCSV()
	case "table":
		formatter = format.NewTable()
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
		writer = output.NewBuffer(h.vim, nvim.Buffer(buf))
	case "yank":
		writer = output.NewYankRegister(h.vim)
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
