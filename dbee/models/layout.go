package models

import "encoding/json"

type (
	LayoutType int
	// Layout is a dict which represents a database structure
	// it's primarely used for the tree view
	Layout struct {
		Name     string     `json:"name"`
		Schema   string     `json:"schema"`
		Database string     `json:"database"`
		Type     LayoutType `json:"type"`
		Children []Layout   `json:"children"`
	}
)

const (
	LayoutNone LayoutType = iota
	LayoutTable
	LayoutHistory
)

func (s LayoutType) String() string {
	switch s {
	case LayoutNone:
		return ""
	case LayoutTable:
		return "table"
	case LayoutHistory:
		return "history"
	default:
		return ""
	}
}

func (s *Layout) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name     string   `json:"name"`
		Schema   string   `json:"schema"`
		Database string   `json:"database"`
		Type     string   `json:"type"`
		Children []Layout `json:"children"`
	}{
		Name:     s.Name,
		Schema:   s.Schema,
		Database: s.Database,
		Type:     s.Type.String(),
		Children: s.Children,
	})
}
