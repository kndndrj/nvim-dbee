package models

import "encoding/json"

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
	// Sort order of children layouts
	ChildrenSortOrder LayoutSortOrder
	// Children layout nodes
	Children []Layout
	// PickItems represents a list of selections (example: database names)
	PickItems []string
}

func (s *Layout) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name              string   `json:"name"`
		Schema            string   `json:"schema"`
		Database          string   `json:"database"`
		Type              string   `json:"type"`
		ChildrenSortOrder string   `json:"children_sort_order"`
		Children          []Layout `json:"children"`
		PickItems         []string `json:"pick_items"`
	}{
		Name:              s.Name,
		Schema:            s.Schema,
		Database:          s.Database,
		Type:              s.Type.String(),
		ChildrenSortOrder: s.ChildrenSortOrder.String(),
		Children:          s.Children,
		PickItems:         s.PickItems,
	})
}
