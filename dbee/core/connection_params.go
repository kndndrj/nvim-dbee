package core

import "encoding/json"

type ConnectionParams struct {
	ID   ConnectionID
	Name string
	Type string
	URL  string
}

// Expand returns a copy of the original parameters with expanded fields
func (p *ConnectionParams) Expand() *ConnectionParams {
	return &ConnectionParams{
		ID:   ConnectionID(expandOrDefault(string(p.ID))),
		Name: expandOrDefault(p.Name),
		Type: expandOrDefault(p.Type),
		URL:  expandOrDefault(p.URL),
	}
}

func (cp *ConnectionParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
		URL  string `json:"url"`
	}{
		ID:   string(cp.ID),
		Name: cp.Name,
		Type: cp.Type,
		URL:  cp.URL,
	})
}
