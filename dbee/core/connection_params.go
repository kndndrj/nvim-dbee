package core

import "encoding/json"

type ConnectionParams struct {
	ID                    ConnectionID
	Name                  string
	Type                  string
	URL                   string
	SSH                   *SSHConfig `json:"ssh,omitempty"`
	UseTunnelInConnection bool       `json:"use_tunnel_in_connection,omitempty"`
}

// SSHConfig represents the SSH tunnel configuration
type SSHConfig struct {
	Host       string   `json:"host" msgpack:"host"`
	User       string   `json:"user" msgpack:"user"`
	Password   string   `json:"password,omitempty" msgpack:"password,omitempty"`
	SSHFile    string   `json:"ssh_file,omitempty" msgpack:"ssh_file,omitempty"`
	LocalPort  int      `json:"local_port" msgpack:"local_port"`
	RemotePort int      `json:"remote_port" msgpack:"remote_port"`
	Options    []string `json:"options,omitempty" msgpack:"options,omitempty"`
}

func (p *ConnectionParams) Expand() *ConnectionParams {
	var sshCopy *SSHConfig
	if p.SSH != nil {
		sshCopy = &SSHConfig{
			Host:       p.SSH.Host,
			User:       p.SSH.User,
			Password:   p.SSH.Password,
			SSHFile:    p.SSH.SSHFile,
			LocalPort:  p.SSH.LocalPort,
			RemotePort: p.SSH.RemotePort,
		}

		if p.SSH.Options != nil {
			sshCopy.Options = make([]string, len(p.SSH.Options))
			copy(sshCopy.Options, p.SSH.Options)
		}
	}

	return &ConnectionParams{
		ID:   ConnectionID(expandOrDefault(string(p.ID))),
		Name: expandOrDefault(p.Name),
		Type: expandOrDefault(p.Type),
		URL:  expandOrDefault(p.URL),
		SSH:  sshCopy,
	}
}

func (cp *ConnectionParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID   string     `json:"id"`
		Name string     `json:"name"`
		Type string     `json:"type"`
		URL  string     `json:"url"`
		SSH  *SSHConfig `json:"ssh,omitempty"`
	}{
		ID:   string(cp.ID),
		Name: cp.Name,
		Type: cp.Type,
		URL:  cp.URL,
		SSH:  cp.SSH,
	})
}
