package handler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

func (h *Handler) storeCallLog() error {
	store := make(map[core.ConnectionID][]*core.Call)

	for connID := range h.lookupConnection {
		calls, err := h.ConnectionGetCalls(connID)
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

	var store map[core.ConnectionID][]*core.Call

	err = decoder.Decode(&store)
	if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	for connID, calls := range store {
		callIDs := make([]core.CallID, len(calls))

		// fill call lookup
		for i, c := range calls {
			h.lookupCall[c.GetID()] = c
			callIDs[i] = c.GetID()
		}

		// add to conn-call lookup
		h.lookupConnectionCall[connID] = append(h.lookupConnectionCall[connID], callIDs...)
	}

	return nil
}
