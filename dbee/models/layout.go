package models

import (
	"github.com/neovim/go-client/msgpack"
)

type LayoutType int

const (
	LayoutTypeNone LayoutType = iota
	LayoutTypeTable
	LayoutTypeHistory
	LayoutTypeDatabaseSwitch
	LayoutTypeView
)

func (s LayoutType) String() string {
	switch s {
	case LayoutTypeNone:
		return ""
	case LayoutTypeTable:
		return "table"
	case LayoutTypeHistory:
		return "history"
	case LayoutTypeDatabaseSwitch:
		return "database_switch"
	case LayoutTypeView:
		return "view"
	default:
		return ""
	}
}

type LayoutSortOrder int

const (
	LayourtSortOrderAscending LayoutSortOrder = iota
	LayourtSortOrderDescending
)

func (s LayoutSortOrder) String() string {
	switch s {
	case LayourtSortOrderAscending:
		return "asc"
	case LayourtSortOrderDescending:
		return "desc"
	default:
		return "asc"
	}
}

// Layout is a dict which represents a database structure
// it's primarely used for the tree view
type Layout struct {
	// Name to be displayed
	Name     string
	Schema   string
	Database string
	// Type of layout
	Type LayoutType
	// Children layout nodes
	Children []Layout
	// PickItems represents a list of selections (example: database names)
	PickItems []string
}

func (l *Layout) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(&struct {
		Name              string   `msgpack:"name"`
		Schema            string   `msgpack:"schema"`
		Database          string   `msgpack:"database"`
		Type              string   `msgpack:"type"`
		Children          []Layout `msgpack:"children"`
		PickItems         []string `msgpack:"pick_items"`
	}{
		Name:      l.Name,
		Schema:    l.Schema,
		Database:  l.Database,
		Type:      l.Type.String(),
		Children:  l.Children,
		PickItems: l.PickItems,
	})
}
