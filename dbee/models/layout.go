package models

import "encoding/json"

type LayoutType int

const (
	LayoutTypeNone LayoutType = iota
	LayoutTypeTable
	LayoutTypeHistory
	LayoutTypeDatabaseSwitch
)

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
	default:
		return ""
	}
}

func (s *Layout) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name      string   `json:"name"`
		Schema    string   `json:"schema"`
		Database  string   `json:"database"`
		Type      string   `json:"type"`
		Children  []Layout `json:"children"`
		PickItems []string `json:"pick_items"`
	}{
		Name:      s.Name,
		Schema:    s.Schema,
		Database:  s.Database,
		Type:      s.Type.String(),
		Children:  s.Children,
		PickItems: s.PickItems,
	})
}
